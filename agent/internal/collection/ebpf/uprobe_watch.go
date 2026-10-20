package ebpf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/thread_koder/mochi/agent/internal/logger"
)

const (
	uprobeScanInterval = time.Second
	uprobeRescanEvery  = 30
)

type fileID struct {
	dev uint64
	ino uint64
}

type pidKey struct {
	pid   uint32
	start uint64
}

func (c *Collector) ensureUprobeWatch() {
	c.uprobeMu.Lock()
	defer c.uprobeMu.Unlock()
	if c.uprobeSeen == nil {
		c.uprobeSeen = make(map[pidKey]struct{})
	}
	if c.uprobeRetry == nil {
		c.uprobeRetry = make(map[pidKey]struct{})
	}
}

func (c *Collector) watchUprobes(ctx context.Context) {
	c.scanUprobes()

	ticker := time.NewTicker(uprobeScanInterval)
	defer ticker.Stop()
	var tick int
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick++
			if tick%uprobeRescanEvery == 0 {
				c.retryFailedUprobes()
			}
			c.scanUprobes()
		}
	}
}

func (c *Collector) scanUprobes() {
	log := logger.WithComponent("ebpf-uprobe")
	entries, err := os.ReadDir("/proc")
	if err != nil {
		log.Error().Err(err).Msg("Failed to read /proc for uprobe attach")
		return
	}

	live := make(map[pidKey]struct{})
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			continue
		}
		start, ok := procStartTime(uint32(pid))
		if !ok {
			continue
		}
		key := pidKey{pid: uint32(pid), start: start}
		live[key] = struct{}{}
		if c.pidSeen(key) {
			continue
		}
		c.inspectMaps(uint32(pid), key)
		c.inspectExe(uint32(pid), key)
		c.markPID(key)
	}

	c.uprobeMu.Lock()
	for key := range c.uprobeSeen {
		if _, ok := live[key]; !ok {
			delete(c.uprobeSeen, key)
			delete(c.uprobeRetry, key)
		}
	}
	c.uprobeMu.Unlock()
}

func (c *Collector) retryFailedUprobes() {
	c.uprobeMu.Lock()
	for key := range c.uprobeRetry {
		delete(c.uprobeSeen, key)
	}
	clear(c.uprobeRetry)
	c.uprobeMu.Unlock()
}

func (c *Collector) inspectMaps(pid uint32, key pidKey) {
	log := logger.WithComponent("ebpf-uprobe")
	mapsData, err := os.ReadFile(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		if c.procUnreadable(err) {
			return
		}
		log.Error().Err(err).Uint32("pid", pid).Msg("Failed to read process maps")
		return
	}
	for line := range bytes.SplitSeq(mapsData, []byte{'\n'}) {
		mapsPath := execMapPath(string(line))
		if mapsPath == "" || !isLibSSL(mapsPath) {
			continue
		}
		c.considerFile(pid, key, procRootPath(pid, mapsPath), mapsPath, false)
	}
}

func (c *Collector) inspectExe(pid uint32, key pidKey) {
	exePath := fmt.Sprintf("/proc/%d/exe", pid)
	c.considerFile(pid, key, exePath, exePath, true)
}

func (c *Collector) considerFile(pid uint32, key pidKey, openPath, label string, allowGo bool) {
	log := logger.WithComponent("ebpf-uprobe")
	tlsLog := logger.WithComponent("ebpf-tls")
	injectLog := logger.WithComponent("ebpf-inject")
	id, err := fileIDOf(openPath)
	if err != nil {
		if processGone(err) {
			c.markRetry(key)
			return
		}
		if errors.Is(err, os.ErrPermission) {
			c.noteProcDenied(err)
			return
		}
		log.Error().Err(err).Str("path", label).Uint32("pid", pid).Msg("Failed to stat uprobe target")
		return
	}
	if allowGo && c.isAgent(id) {
		c.storeTLSSkip(id)
		c.storeInjectSkip(id)
		return
	}

	needTLS := c.tlsEnabled && !c.knownTLSFile(id)
	needInject := c.injectEnabled && !c.knownInjectFile(id)
	if !needTLS && !needInject {
		return
	}

	var goRETs map[string]goRETInfo
	if allowGo {
		var loadErr error
		goRETs, loadErr = loadGoRETs(openPath, goTLSSymbolNames)
		if loadErr != nil {
			if processGone(loadErr) {
				c.markRetry(key)
				return
			}
			if errors.Is(loadErr, os.ErrPermission) {
				c.noteProcDenied(loadErr)
				return
			}
			c.markRetry(key)
			log.Error().Err(loadErr).Str("path", label).Uint32("pid", pid).Msg("Failed to decode Go TLS RETs")
			return
		}
	}

	gone := false
	if needTLS {
		links, hit, err := c.attachTLSFile(openPath, allowGo, goRETs)
		if err != nil {
			if processGone(err) {
				gone = true
				c.markRetry(key)
			} else if errors.Is(err, os.ErrPermission) {
				c.noteProcDenied(err)
				return
			} else {
				c.markRetry(key)
				tlsLog.Error().Err(err).Str("path", label).Uint32("pid", pid).Msg("Failed to attach TLS probes")
			}
		} else if !hit {
			c.storeTLSSkip(id)
		} else if !c.storeTLSLinks(id, links) {
			err := closeLinks(links)
			if err != nil {
				tlsLog.Error().Err(err).Str("path", label).Msg("Failed to close duplicate TLS probes")
			}
		}
	}

	if gone || !needInject {
		return
	}
	links, hit, err := c.attachInjectFile(openPath, allowGo, goRETs)
	if err != nil {
		if processGone(err) {
			c.markRetry(key)
			return
		}
		if errors.Is(err, os.ErrPermission) {
			c.noteProcDenied(err)
			return
		}
		c.markRetry(key)
		injectLog.Error().Err(err).Str("path", label).Uint32("pid", pid).Msg("Failed to attach inject TLS probes")
		return
	}
	if !hit {
		c.storeInjectSkip(id)
		return
	}
	if !c.storeInjectLinks(id, links) {
		err := closeLinks(links)
		if err != nil {
			injectLog.Error().Err(err).Str("path", label).Msg("Failed to close duplicate inject TLS probes")
		}
		return
	}
}

func (c *Collector) noteProcDenied(err error) {
	log := logger.WithComponent("ebpf-uprobe")
	c.uprobeMu.Lock()
	defer c.uprobeMu.Unlock()
	if c.uprobeDenied {
		return
	}
	c.uprobeDenied = true
	log.Error().Err(err).Msg("Uprobe discovery cannot read process maps or exe")
}

func (c *Collector) pidSeen(key pidKey) bool {
	c.uprobeMu.Lock()
	defer c.uprobeMu.Unlock()
	_, ok := c.uprobeSeen[key]
	return ok
}

func (c *Collector) markPID(key pidKey) {
	c.uprobeMu.Lock()
	defer c.uprobeMu.Unlock()
	if c.uprobeStop {
		return
	}
	c.uprobeSeen[key] = struct{}{}
}

func (c *Collector) markRetry(key pidKey) {
	c.uprobeMu.Lock()
	defer c.uprobeMu.Unlock()
	if c.uprobeStop {
		return
	}
	c.uprobeRetry[key] = struct{}{}
}

func (c *Collector) isAgent(id fileID) bool {
	c.uprobeMu.Lock()
	defer c.uprobeMu.Unlock()
	return c.agentFile != nil && *c.agentFile == id
}

// procUnreadable is true when the caller should stop without a per call error
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

func processGone(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH)
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

func procStartTime(pid uint32) (uint64, bool) {
	statData, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, false
	}
	end := bytes.LastIndexByte(statData, ')')
	if end < 0 || end+2 >= len(statData) {
		return 0, false
	}
	fields := strings.Fields(string(statData[end+2:]))
	// starttime is stat field 22. After comm it is fields[19].
	if len(fields) < 20 {
		return 0, false
	}
	start, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return 0, false
	}
	return start, true
}

func procRootPath(pid uint32, mapsPath string) string {
	return fmt.Sprintf("/proc/%d/root%s", pid, mapsPath)
}

func execMapPath(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 6 || !strings.Contains(fields[1], "x") {
		return ""
	}
	mapsPath := strings.Join(fields[5:], " ")
	mapsPath = strings.TrimSuffix(mapsPath, " (deleted)")
	if mapsPath == "" || mapsPath[0] != '/' {
		return ""
	}
	return mapsPath
}

func isLibSSL(mapsPath string) bool {
	base := path.Base(mapsPath)
	return base == "libssl.so" || strings.HasPrefix(base, "libssl.so.")
}
