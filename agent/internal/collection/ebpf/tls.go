package ebpf

import (
	"context"
	"errors"
	"io"

	"github.com/cilium/ebpf/ringbuf"
	"github.com/thread_koder/mochi/agent/internal/collection/l7"
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

var opensslHooks = []uprobeHook{
	{"SSL_write", tlsCookieSSLWrite, false},
	{"SSL_read", tlsCookieSSLRead, false},
	{"SSL_write_ex", tlsCookieSSLWriteEx, false},
	{"SSL_read_ex", tlsCookieSSLReadEx, false},
}

var goTLSHooks = []uprobeHook{
	{"crypto/tls.(*Conn).Write", tlsCookieGoWrite, true},
	{"crypto/tls.(*Conn).Read", tlsCookieGoRead, false},
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
		c.handleTLSRecord(record.RawSample)
	}
}

func (c *Collector) handleTLSRecord(raw []byte) {
	event, err := parseStreamWireEvent(raw)
	if err != nil {
		return
	}
	switch event.Kind {
	case l7.KindOpenSSL, l7.KindGoTLS:
	default:
		return
	}
	c.feedL7(event)
}

func (c *Collector) knownTLSFile(id fileID) bool {
	c.uprobeMu.Lock()
	defer c.uprobeMu.Unlock()
	if c.uprobeStop {
		return true
	}
	if c.tlsInodes == nil {
		return true
	}
	if _, ok := c.tlsInodes[id]; ok {
		return true
	}
	_, ok := c.tlsSkip[id]
	return ok
}

func (c *Collector) storeTLSSkip(id fileID) {
	c.uprobeMu.Lock()
	defer c.uprobeMu.Unlock()
	if c.uprobeStop || c.tlsSkip == nil {
		return
	}
	c.tlsSkip[id] = struct{}{}
}

func (c *Collector) storeTLSLinks(id fileID, links []io.Closer) bool {
	c.uprobeMu.Lock()
	defer c.uprobeMu.Unlock()
	if c.uprobeStop || c.tlsInodes == nil {
		return false
	}
	if _, ok := c.tlsInodes[id]; ok {
		return false
	}
	c.tlsInodes[id] = links
	return true
}

func (c *Collector) closeTLS() error {
	c.uprobeMu.Lock()
	var uprobes []io.Closer
	for _, inodeLinks := range c.tlsInodes {
		uprobes = append(uprobes, inodeLinks...)
	}
	c.tlsInodes = nil
	c.tlsSkip = nil
	kernel := c.tlsLinks
	c.tlsLinks = nil
	events := c.tlsEvents
	c.tlsEvents = nil
	c.uprobeMu.Unlock()

	var err error
	if events != nil {
		err = errors.Join(err, events.Close())
	}
	err = errors.Join(err, closeLinks(kernel), closeLinks(uprobes))
	if c.tlsEnabled {
		err = errors.Join(err, c.tlsObjs.Close())
	}
	c.tlsEnabled = false
	return err
}

func (c *Collector) attachTLSFile(openPath string, allowGo bool, goRETs map[string]goRETInfo) ([]io.Closer, bool, error) {
	return c.attachUprobeFile(
		openPath,
		allowGo,
		goRETs,
		opensslHooks,
		goTLSHooks,
		c.tlsObjs.MochiTlsEnter,
		c.tlsObjs.MochiTlsRet,
		c.tlsObjs.MochiTlsEnter,
		c.tlsObjs.MochiTlsRet,
		tlsCookieGoRet,
		true,
	)
}
