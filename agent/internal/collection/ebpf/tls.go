package ebpf

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/rs/zerolog"
	"github.com/thread_koder/mochi/agent/internal/logger"
)

// Must match TLS_COOKIE_* in bpf/tls_plain.c.
const (
	tlsCookieSSLWrite   = 1
	tlsCookieSSLRead    = 2
	tlsCookieSSLWriteEx = 3
	tlsCookieSSLReadEx  = 4
	tlsCookieGoWrite    = 5
	tlsCookieGoRead     = 6
	tlsCookieGoRet      = 7
)

type fileID struct {
	dev uint64
	ino uint64
}

type pidKey struct {
	pid   uint32
	start uint64
}

type tlsHook struct {
	symbol string
	cookie uint64
}

func (c *Collector) runTLS(ctx context.Context) {
	log := logger.WithComponent("ebpf-tls")
	go func() {
		<-ctx.Done()
		_ = c.tlsEvents.Close()
	}()

	for {
		record, err := c.tlsEvents.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				return
			}
			log.Error().Err(err).Msg("Failed to read TLS ringbuf")
			continue
		}
		c.handleTLSRecord(log, record.RawSample)
	}
}

func (c *Collector) handleTLSRecord(log zerolog.Logger, raw []byte) {
	if zerolog.GlobalLevel() > zerolog.DebugLevel {
		return
	}

	event, err := parseStreamWireEvent(raw)
	if err != nil {
		return
	}
	via, ok := tlsVia(event.Kind)
	if !ok {
		return
	}
	c.dumpHTTP1(log, "TLS plaintext", via, event)
}

func tlsVia(kind uint8) (string, bool) {
	switch kind {
	case streamKindOpenSSL:
		return "openssl", true
	case streamKindGoTLS:
		return "gotls", true
	default:
		return "", false
	}
}

func (c *Collector) noteProcDenied(err error) {
	log := logger.WithComponent("ebpf-tls")
	c.tlsMu.Lock()
	defer c.tlsMu.Unlock()
	if c.tlsDenied {
		return
	}
	c.tlsDenied = true
	log.Error().Err(err).Msg("TLS discovery cannot read process maps or exe")
}

func (c *Collector) pidSeen(key pidKey) bool {
	c.tlsMu.Lock()
	defer c.tlsMu.Unlock()
	_, ok := c.tlsSeen[key]
	return ok
}

func (c *Collector) markPID(key pidKey) {
	c.tlsMu.Lock()
	defer c.tlsMu.Unlock()
	if c.tlsStop {
		return
	}
	c.tlsSeen[key] = struct{}{}
}

func (c *Collector) markRetry(key pidKey) {
	c.tlsMu.Lock()
	defer c.tlsMu.Unlock()
	if c.tlsStop {
		return
	}
	c.tlsRetry[key] = struct{}{}
}

func (c *Collector) knownFile(id fileID) bool {
	c.tlsMu.Lock()
	defer c.tlsMu.Unlock()
	if c.tlsStop {
		return true
	}
	if _, ok := c.tlsInodes[id]; ok {
		return true
	}
	_, ok := c.tlsSkip[id]
	return ok
}

func (c *Collector) isAgent(id fileID) bool {
	c.tlsMu.Lock()
	defer c.tlsMu.Unlock()
	return c.agentFile != nil && *c.agentFile == id
}

func (c *Collector) storeSkip(id fileID) {
	c.tlsMu.Lock()
	defer c.tlsMu.Unlock()
	if c.tlsStop {
		return
	}
	c.tlsSkip[id] = struct{}{}
}

func (c *Collector) storeLinks(id fileID, links []io.Closer) bool {
	c.tlsMu.Lock()
	defer c.tlsMu.Unlock()
	if c.tlsStop {
		return false
	}
	if _, ok := c.tlsInodes[id]; ok {
		return false
	}
	c.tlsInodes[id] = links
	return true
}

func (c *Collector) closeTLS() error {
	c.tlsMu.Lock()
	c.tlsStop = true
	var uprobes []io.Closer
	for _, inodeLinks := range c.tlsInodes {
		uprobes = append(uprobes, inodeLinks...)
	}
	c.tlsInodes = nil
	kernel := c.tlsLinks
	c.tlsLinks = nil
	events := c.tlsEvents
	c.tlsEvents = nil
	c.tlsMu.Unlock()

	var err error
	if events != nil {
		err = errors.Join(err, events.Close())
	}
	err = errors.Join(err, closeLinks(kernel), closeLinks(uprobes))
	if c.tlsEnabled {
		err = errors.Join(err, c.tlsObjs.Close())
	}
	return err
}

func missingSymbol(err error) bool {
	return errors.Is(err, link.ErrNoSymbol) || errors.Is(err, link.ErrNotSupported)
}

func processGone(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH)
}

// procUnreadable is true when the caller should stop without a per-call error
// log. Permission denial is logged once via noteProcDenied.
func (c *Collector) procUnreadable(err error) bool {
	if processGone(err) {
		return true
	}
	if errors.Is(err, os.ErrPermission) {
		c.noteProcDenied(err)
		return true
	}
	return false
}

func fileIDOf(path string) (fileID, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fileID{}, err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileID{}, fmt.Errorf("stat %s: no Stat_t", path)
	}
	return fileID{dev: st.Dev, ino: st.Ino}, nil
}
