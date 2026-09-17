package ebpf

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/thread_koder/mochi/agent/internal/logger"
)

const (
	tlsScanInterval = time.Second
	tlsRescanEvery  = 30
)

func (c *Collector) watchTLS(ctx context.Context) {
	c.scanTLS()

	ticker := time.NewTicker(tlsScanInterval)
	defer ticker.Stop()
	var tick int
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick++
			if tick%tlsRescanEvery == 0 {
				c.retryFailedTLS()
			}
			c.scanTLS()
		}
	}
}

func (c *Collector) scanTLS() {
	log := logger.WithComponent("ebpf-tls")
	entries, err := os.ReadDir("/proc")
	if err != nil {
		log.Error().Err(err).Msg("Failed to read /proc for TLS probes")
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

	c.tlsMu.Lock()
	for key := range c.tlsSeen {
		if _, ok := live[key]; !ok {
			delete(c.tlsSeen, key)
			delete(c.tlsRetry, key)
		}
	}
	c.tlsMu.Unlock()
}

func (c *Collector) retryFailedTLS() {
	c.tlsMu.Lock()
	for key := range c.tlsRetry {
		delete(c.tlsSeen, key)
	}
	clear(c.tlsRetry)
	c.tlsMu.Unlock()
}

func (c *Collector) inspectMaps(pid uint32, key pidKey) {
	log := logger.WithComponent("ebpf-tls")
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		if c.procUnreadable(err) {
			return
		}
		log.Error().Err(err).Uint32("pid", pid).Msg("Failed to read process maps")
		return
	}
	for line := range bytes.SplitSeq(raw, []byte{'\n'}) {
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
	log := logger.WithComponent("ebpf-tls")
	id, err := fileIDOf(openPath)
	if err != nil {
		if c.procUnreadable(err) {
			return
		}
		log.Error().Err(err).Str("path", label).Uint32("pid", pid).Msg("Failed to stat TLS target")
		return
	}
	if allowGo && c.isAgent(id) {
		c.storeSkip(id)
		return
	}
	if c.knownFile(id) {
		return
	}

	links, hit, err := c.attachFile(openPath, allowGo)
	if err != nil {
		c.markRetry(key)
		log.Error().Err(err).Str("path", label).Uint32("pid", pid).Msg("Failed to attach TLS probes")
		return
	}
	if !hit {
		c.storeSkip(id)
		return
	}
	if !c.storeLinks(id, links) {
		err := closeLinks(links)
		if err != nil {
			log.Error().Err(err).Str("path", label).Msg("Failed to close duplicate TLS probes")
		}
		return
	}
	log.Debug().Str("path", label).Int("links", len(links)).Msg("Attached TLS probes")
}

func procStartTime(pid uint32) (uint64, bool) {
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, false
	}
	end := bytes.LastIndexByte(raw, ')')
	if end < 0 || end+2 >= len(raw) {
		return 0, false
	}
	fields := strings.Fields(string(raw[end+2:]))
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
