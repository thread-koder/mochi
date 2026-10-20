package ebpf

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/thread_koder/mochi/agent/internal/collection/l7"
	"github.com/thread_koder/mochi/agent/internal/logger"
	"github.com/thread_koder/mochi/agent/internal/span"
	"golang.org/x/sys/unix"
)

const (
	tpSampledFlag = 0x01
	cgroupFSRoot  = "/sys/fs/cgroup"

	// Must match INJECT_COOKIE_* in bpf/inject.c.
	injectCookieGoWrite   = 1
	injectCookieGoRead    = 2
	injectCookieGoRet     = 3
	injectCookieSSLRead   = 4
	injectCookieSSLReadEx = 5
)

var injectGoHooks = []uprobeHook{
	{"crypto/tls.(*Conn).Write", injectCookieGoWrite, true},
	{"crypto/tls.(*Conn).Read", injectCookieGoRead, false},
}

var injectOpenSSLReadHooks = []uprobeHook{
	{"SSL_read", injectCookieSSLRead, false},
	{"SSL_read_ex", injectCookieSSLReadEx, false},
}

func (c *Collector) loadInject(sampleRate float64) error {
	log := logger.WithComponent("ebpf-inject")

	if err := loadInjectObjects(&c.injectObjs, nil); err != nil {
		return fmt.Errorf("load inject eBPF objects: %w", err)
	}

	if err := c.writeInjectSampleThreshold(sampleRate); err != nil {
		closeErr := c.injectObjs.Close()
		return errors.Join(fmt.Errorf("write inject sample threshold: %w", err), closeErr)
	}

	cgroupPath, err := findCgroupV2Path()
	if err != nil {
		closeErr := c.injectObjs.Close()
		return errors.Join(fmt.Errorf("find cgroup v2: %w", err), closeErr)
	}

	sockopsLink, err := link.AttachCgroup(link.CgroupOptions{
		Path:    cgroupPath,
		Attach:  ebpf.AttachCGroupSockOps,
		Program: c.injectObjs.MochiInjectSockops,
	})
	if err != nil {
		closeErr := c.injectObjs.Close()
		return errors.Join(fmt.Errorf("attach cgroup sockops: %w", err), closeErr)
	}

	if err := link.RawAttachProgram(link.RawAttachProgramOptions{
		Target:  c.injectObjs.InjectSocks.FD(),
		Program: c.injectObjs.MochiInjectSkMsg,
		Attach:  ebpf.AttachSkMsgVerdict,
	}); err != nil {
		closeErr := errors.Join(sockopsLink.Close(), c.injectObjs.Close())
		return errors.Join(fmt.Errorf("attach sk_msg: %w", err), closeErr)
	}
	c.injectSkMsgAttached = true

	var links []io.Closer
	links = append(links, sockopsLink)
	for _, step := range []struct {
		prog *ebpf.Program
		name string
	}{
		{c.injectObjs.MochiInjectRecvEnter, "fentry tcp_recvmsg inject"},
		{c.injectObjs.MochiInjectRecvExit, "fexit tcp_recvmsg inject"},
		{c.injectObjs.MochiInjectTxEnter, "fentry tcp_sendmsg_locked inject"},
	} {
		if err := attachTracing(&links, step.prog, step.name); err != nil {
			closeErr := errors.Join(
				c.detachInjectSkMsg(),
				closeLinks(links),
				c.injectObjs.Close(),
			)
			return errors.Join(err, closeErr)
		}
	}

	if err := c.seedInjectSocks(); err != nil {
		log.Error().Err(err).Msg("Failed to seed inject sockets from iter/tcp. Continuing with sockops-only insert")
	}

	c.writeUserAllowed = bpfProbeWriteUserAllowed()
	if !c.writeUserAllowed {
		log.Info().Msg("Kernel lockdown blocks bpf_probe_write_user. Continuing without TLS Traceparent inject")
	}

	c.ensureAgentFile(log, "Go inject probes may include the agent")
	c.ensureUprobeWatch()
	c.uprobeMu.Lock()
	if c.injectInodes == nil {
		c.injectInodes = make(map[fileID][]io.Closer)
	}
	if c.injectSkip == nil {
		c.injectSkip = make(map[fileID]struct{})
	}
	c.uprobeMu.Unlock()
	c.injectLinks = links
	c.injectEnabled = true
	log.Info().Msg("eBPF Traceparent inject loaded")
	return nil
}

func (c *Collector) writeInjectSampleThreshold(sampleRate float64) error {
	var threshold uint64
	switch {
	case sampleRate <= 0:
		threshold = 0
	case sampleRate >= 1:
		threshold = 1 << 63
	default:
		threshold = uint64(sampleRate * float64(uint64(1)<<63))
	}
	key := uint32(0)
	cfg := injectInjectCfg{SampleThreshold: threshold}
	return c.injectObjs.InjectCfg.Put(key, cfg)
}

func (c *Collector) seedInjectSocks() error {
	iterLink, err := link.AttachIter(link.IterOptions{
		Program: c.injectObjs.MochiInjectSockIter,
	})
	if err != nil {
		return fmt.Errorf("attach iter/tcp: %w", err)
	}
	defer iterLink.Close()

	reader, err := iterLink.Open()
	if err != nil {
		return fmt.Errorf("open iter/tcp: %w", err)
	}
	defer reader.Close()

	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("run iter/tcp: %w", err)
	}
	return nil
}

func (c *Collector) detachInjectSkMsg() error {
	if !c.injectSkMsgAttached {
		return nil
	}
	err := link.RawDetachProgram(link.RawDetachProgramOptions{
		Target:  c.injectObjs.InjectSocks.FD(),
		Program: c.injectObjs.MochiInjectSkMsg,
		Attach:  ebpf.AttachSkMsgVerdict,
	})
	c.injectSkMsgAttached = false
	return err
}

func (c *Collector) closeInject() error {
	c.uprobeMu.Lock()
	var uprobes []io.Closer
	for _, inodeLinks := range c.injectInodes {
		uprobes = append(uprobes, inodeLinks...)
	}
	c.injectInodes = nil
	c.injectSkip = nil
	kernel := c.injectLinks
	c.injectLinks = nil
	c.uprobeMu.Unlock()

	var err error
	err = errors.Join(err, c.detachInjectSkMsg())
	err = errors.Join(err, closeLinks(kernel))
	err = errors.Join(err, closeLinks(uprobes))
	err = errors.Join(err, c.injectObjs.Close())
	c.injectEnabled = false
	return err
}

// TakeOutgoing adopts IDs written by inject for this outbound hop.
// LookupAndDelete so extract cannot remint against the same wire IDs.
func (c *Collector) TakeOutgoing(pid uint32, sport, dport uint16, stream uint32) (l7.InjectedIDs, bool) {
	if !c.injectEnabled {
		return l7.InjectedIDs{}, false
	}
	key := injectEgressKey{Pid: pid, Sport: sport, Dport: dport, Stream: stream}
	var tp injectTpIds
	if err := c.injectObjs.OutgoingTp.LookupAndDelete(&key, &tp); err != nil {
		return l7.InjectedIDs{}, false
	}
	var ids l7.InjectedIDs
	copy(ids.TraceID[:], tp.TraceId[:])
	copy(ids.SpanID[:], tp.SpanId[:])
	copy(ids.ParentSpanID[:], tp.ParentId[:])
	if span.IsZeroTraceID(ids.TraceID) || span.IsZeroSpanID(ids.SpanID) {
		return l7.InjectedIDs{}, false
	}
	ids.Sampled = tp.Flags&tpSampledFlag != 0
	return ids, true
}

func (c *Collector) attachInjectFile(openPath string, allowGo bool, goRETs map[string]goRETInfo) ([]io.Closer, bool, error) {
	return c.attachUprobeFile(
		openPath,
		allowGo,
		goRETs,
		injectOpenSSLReadHooks,
		injectGoHooks,
		c.injectObjs.MochiInjectSslEnter,
		c.injectObjs.MochiInjectSslRet,
		c.injectObjs.MochiInjectGoEnter,
		c.injectObjs.MochiInjectGoRet,
		injectCookieGoRet,
		c.writeUserAllowed,
	)
}

func (c *Collector) knownInjectFile(id fileID) bool {
	c.uprobeMu.Lock()
	defer c.uprobeMu.Unlock()
	if c.uprobeStop {
		return true
	}
	if c.injectInodes == nil {
		return true
	}
	if _, ok := c.injectInodes[id]; ok {
		return true
	}
	_, ok := c.injectSkip[id]
	return ok
}

func (c *Collector) storeInjectSkip(id fileID) {
	c.uprobeMu.Lock()
	defer c.uprobeMu.Unlock()
	if c.uprobeStop || c.injectSkip == nil {
		return
	}
	c.injectSkip[id] = struct{}{}
}

func (c *Collector) storeInjectLinks(id fileID, links []io.Closer) bool {
	c.uprobeMu.Lock()
	defer c.uprobeMu.Unlock()
	if c.uprobeStop || c.injectInodes == nil {
		return false
	}
	if _, ok := c.injectInodes[id]; ok {
		return false
	}
	c.injectInodes[id] = links
	return true
}

func bpfProbeWriteUserAllowed() bool {
	lockdown, err := os.ReadFile("/sys/kernel/security/lockdown")
	if err != nil {
		return true
	}
	return strings.Contains(string(lockdown), "[none]")
}

func findCgroupV2Path() (string, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(cgroupFSRoot, &st); err != nil {
		return "", err
	}
	if st.Type == unix.CGROUP2_SUPER_MAGIC {
		return cgroupFSRoot, nil
	}
	unified := filepath.Join(cgroupFSRoot, "unified")
	if err := syscall.Statfs(unified, &st); err != nil {
		return "", fmt.Errorf("cgroup v2 not mounted at %s or %s", cgroupFSRoot, unified)
	}
	if st.Type != unix.CGROUP2_SUPER_MAGIC {
		return "", fmt.Errorf("cgroup v2 not mounted at %s or %s", cgroupFSRoot, unified)
	}
	return unified, nil
}
