package ebpf

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
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
)

func (c *Collector) loadHTTP1Inject(sampleRate float64) error {
	log := logger.WithComponent("ebpf-inject-http1")

	if err := loadHttp1injectObjects(&c.injectObjs, nil); err != nil {
		return fmt.Errorf("load HTTP/1 inject eBPF objects: %w", err)
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
		log.Error().Err(err).Msg("Failed to seed inject_socks from iter/tcp. Continuing with sockops-only insert")
	}

	c.injectLinks = links
	c.injectEnabled = true
	log.Info().Msg("eBPF HTTP/1 Traceparent inject loaded")
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
	cfg := http1injectInjectCfg{SampleThreshold: threshold}
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

func (c *Collector) closeHTTP1Inject() error {
	var err error
	err = errors.Join(err, c.detachInjectSkMsg())
	err = errors.Join(err, closeLinks(c.injectLinks))
	err = errors.Join(err, c.injectObjs.Close())
	c.injectEnabled = false
	return err
}

// TakeOutgoing adopts IDs written by sk_msg inject for this outbound hop.
// Deletes the map entry so a later extract cannot remint against the wire.
func (c *Collector) TakeOutgoing(pid uint32, sport, dport uint16) (l7.InjectedIDs, bool) {
	if !c.injectEnabled || c.injectObjs.OutgoingTp == nil {
		return l7.InjectedIDs{}, false
	}
	key := http1injectEgressKey{Pid: pid, Sport: sport, Dport: dport}
	var tp http1injectTpIds
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
