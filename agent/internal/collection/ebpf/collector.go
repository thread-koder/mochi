package ebpf

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/thread_koder/mochi/agent/internal/collection/aggregate"
	"github.com/thread_koder/mochi/agent/internal/collection/conntrack"
	"github.com/thread_koder/mochi/agent/internal/collection/dns"
	"github.com/thread_koder/mochi/agent/internal/collection/http1"
	"github.com/thread_koder/mochi/agent/internal/collection/identity"
	"github.com/thread_koder/mochi/agent/internal/logger"
)

// ServerPorts marks local UDP ports where sendmsg is server-side (bound/unconnected).
// Only the UDP fexit hook consults this. TCP state transitions are client-shaped.
type ServerPorts interface {
	IsBound(podUID string, port uint16) bool
}

// Collector owns loaded eBPF programs and event loops for TCP, UDP, DNS, the
// plaintext TCP byte stream, and TLS plaintext uprobes.
type Collector struct {
	tcpObjs    tcpstateObjects
	tcpLink    link.Link
	tcpEvents  *ringbuf.Reader
	tcpEnabled bool

	udpObjs    udpflowObjects
	udpLinks   []io.Closer
	udpEvents  *ringbuf.Reader
	udpEnabled bool

	dnsObjs    dnsrecvObjects
	dnsLinks   []io.Closer
	dnsEvents  *ringbuf.Reader
	dnsEnabled bool

	streamObjs    tcpstreamObjects
	streamLinks   []io.Closer
	streamEvents  *ringbuf.Reader
	streamEnabled bool

	tlsObjs    tlsplainObjects
	tlsLinks   []io.Closer
	tlsEvents  *ringbuf.Reader
	tlsEnabled bool

	tlsMu     sync.Mutex
	tlsStop   bool
	tlsDenied bool
	tlsInodes map[fileID][]io.Closer
	tlsSkip   map[fileID]struct{}
	tlsSeen   map[pidKey]struct{}
	tlsRetry  map[pidKey]struct{}
	agentFile *fileID

	store           *aggregate.Store
	resolver        *identity.Resolver
	conntrackClient *conntrack.Client
	serverPorts     ServerPorts
	dnsCache        *dns.Cache
	http1           *http1.Tracker
}

func Load(
	store *aggregate.Store,
	resolver *identity.Resolver,
	conntrackClient *conntrack.Client,
	serverPorts ServerPorts,
	dnsCache *dns.Cache,
	http1Tracker *http1.Tracker,
) (*Collector, error) {
	log := logger.WithComponent("ebpf")

	if _, err := os.Stat("/sys/kernel/btf/vmlinux"); err != nil {
		return nil, fmt.Errorf("kernel BTF not available at /sys/kernel/btf/vmlinux: %w", err)
	}

	if err := rlimit.RemoveMemlock(); err != nil {
		log.Debug().Err(err).Msg("RemoveMemlock failed. Continuing")
	}

	collector := &Collector{
		store:           store,
		resolver:        resolver,
		conntrackClient: conntrackClient,
		serverPorts:     serverPorts,
		dnsCache:        dnsCache,
		http1:           http1Tracker,
	}

	if err := collector.loadTCP(); err != nil {
		log.Error().Err(err).Msg("TCP eBPF load failed. Continuing without TCP")
	}
	if err := collector.loadUDP(); err != nil {
		log.Error().Err(err).Msg("UDP eBPF load failed. Continuing without UDP")
	}
	if err := collector.loadDNS(); err != nil {
		log.Error().Err(err).Msg("DNS eBPF load failed. Continuing without DNS correlation")
	}
	if err := collector.loadStream(); err != nil {
		log.Error().Err(err).Msg("TCP stream eBPF load failed. Continuing without plaintext stream")
	}
	if err := collector.loadTLS(); err != nil {
		log.Error().Err(err).Msg("TLS eBPF load failed. Continuing without TLS plaintext")
	}
	if !collector.tcpEnabled && !collector.udpEnabled {
		err := errors.Join(
			fmt.Errorf("no TCP or UDP eBPF programs loaded"),
			collector.Close(),
		)
		return nil, err
	}

	return collector, nil
}

func (c *Collector) loadTCP() error {
	log := logger.WithComponent("ebpf-tcp")

	if err := loadTcpstateObjects(&c.tcpObjs, nil); err != nil {
		return fmt.Errorf("load TCP eBPF objects: %w", err)
	}

	stateLink, err := link.Tracepoint("sock", "inet_sock_set_state", c.tcpObjs.MochiInetSockSetState, nil)
	if err != nil {
		closeErr := c.tcpObjs.Close()
		return errors.Join(fmt.Errorf("attach inet_sock_set_state: %w", err), closeErr)
	}

	events, err := ringbuf.NewReader(c.tcpObjs.Events)
	if err != nil {
		closeErr := errors.Join(stateLink.Close(), c.tcpObjs.Close())
		return errors.Join(fmt.Errorf("open TCP ringbuf: %w", err), closeErr)
	}

	c.tcpLink = stateLink
	c.tcpEvents = events
	c.tcpEnabled = true
	log.Info().Msg("eBPF TCP state collector loaded")
	return nil
}

func (c *Collector) loadUDP() error {
	log := logger.WithComponent("ebpf-udp")

	if err := loadUdpflowObjects(&c.udpObjs, nil); err != nil {
		return fmt.Errorf("load UDP eBPF objects: %w", err)
	}

	var links []io.Closer
	for _, step := range []struct {
		prog *ebpf.Program
		name string
	}{
		{c.udpObjs.MochiUdpSendmsg, "fexit udp_sendmsg"},
		{c.udpObjs.MochiUdpv6Sendmsg, "fexit udpv6_sendmsg"},
		{c.udpObjs.MochiUdpRecvmsg, "fexit udp_recvmsg"},
		{c.udpObjs.MochiUdpv6Recvmsg, "fexit udpv6_recvmsg"},
	} {
		if err := attachTracing(&links, step.prog, step.name); err != nil {
			closeErr := errors.Join(closeLinks(links), c.udpObjs.Close())
			return errors.Join(err, closeErr)
		}
	}

	events, err := ringbuf.NewReader(c.udpObjs.OpenEvents)
	if err != nil {
		closeErr := errors.Join(closeLinks(links), c.udpObjs.Close())
		return errors.Join(fmt.Errorf("open UDP ringbuf: %w", err), closeErr)
	}

	c.udpLinks = links
	c.udpEvents = events
	c.udpEnabled = true
	log.Info().Msg("eBPF UDP flow collector loaded")
	return nil
}

func (c *Collector) loadDNS() error {
	log := logger.WithComponent("ebpf-dns")

	if err := loadDnsrecvObjects(&c.dnsObjs, nil); err != nil {
		return fmt.Errorf("load DNS eBPF objects: %w", err)
	}

	var links []io.Closer
	for _, step := range []struct {
		prog *ebpf.Program
		name string
	}{
		{c.dnsObjs.MochiUdpRecvmsgEnter, "fentry udp_recvmsg"},
		{c.dnsObjs.MochiUdpRecvmsgExit, "fexit udp_recvmsg"},
		{c.dnsObjs.MochiUdpv6RecvmsgEnter, "fentry udpv6_recvmsg"},
		{c.dnsObjs.MochiUdpv6RecvmsgExit, "fexit udpv6_recvmsg"},
		{c.dnsObjs.MochiTcpRecvmsgEnter, "fentry tcp_recvmsg"},
		{c.dnsObjs.MochiTcpRecvmsgExit, "fexit tcp_recvmsg"},
	} {
		if err := attachTracing(&links, step.prog, step.name); err != nil {
			closeErr := errors.Join(closeLinks(links), c.dnsObjs.Close())
			return errors.Join(err, closeErr)
		}
	}

	events, err := ringbuf.NewReader(c.dnsObjs.Events)
	if err != nil {
		closeErr := errors.Join(closeLinks(links), c.dnsObjs.Close())
		return errors.Join(fmt.Errorf("open DNS ringbuf: %w", err), closeErr)
	}

	c.dnsLinks = links
	c.dnsEvents = events
	c.dnsEnabled = true
	log.Info().Msg("eBPF DNS response collector loaded")
	return nil
}

func (c *Collector) loadStream() error {
	log := logger.WithComponent("ebpf-stream")

	if err := loadTcpstreamObjects(&c.streamObjs, nil); err != nil {
		return fmt.Errorf("load TCP stream eBPF objects: %w", err)
	}

	var links []io.Closer
	for _, step := range []struct {
		prog *ebpf.Program
		name string
	}{
		{c.streamObjs.MochiStreamSendEnter, "fentry tcp_sendmsg"},
		{c.streamObjs.MochiStreamSendExit, "fexit tcp_sendmsg"},
		{c.streamObjs.MochiStreamRecvEnter, "fentry tcp_recvmsg"},
		{c.streamObjs.MochiStreamRecvExit, "fexit tcp_recvmsg"},
	} {
		if err := attachTracing(&links, step.prog, step.name); err != nil {
			closeErr := errors.Join(closeLinks(links), c.streamObjs.Close())
			return errors.Join(err, closeErr)
		}
	}

	events, err := ringbuf.NewReader(c.streamObjs.Events)
	if err != nil {
		closeErr := errors.Join(closeLinks(links), c.streamObjs.Close())
		return errors.Join(fmt.Errorf("open TCP stream ringbuf: %w", err), closeErr)
	}

	c.streamLinks = links
	c.streamEvents = events
	c.streamEnabled = true
	log.Info().Msg("eBPF TCP stream collector loaded")
	return nil
}

func (c *Collector) loadTLS() error {
	log := logger.WithComponent("ebpf-tls")

	if err := loadTlsplainObjects(&c.tlsObjs, nil); err != nil {
		return fmt.Errorf("load TLS eBPF objects: %w", err)
	}

	var links []io.Closer
	for _, step := range []struct {
		prog *ebpf.Program
		name string
	}{
		{c.tlsObjs.MochiTlsSendEnter, "fentry tcp_sendmsg"},
		{c.tlsObjs.MochiTlsRecvEnter, "fentry tcp_recvmsg"},
	} {
		if err := attachTracing(&links, step.prog, step.name); err != nil {
			closeErr := errors.Join(closeLinks(links), c.tlsObjs.Close())
			return errors.Join(err, closeErr)
		}
	}

	events, err := ringbuf.NewReader(c.tlsObjs.Events)
	if err != nil {
		closeErr := errors.Join(closeLinks(links), c.tlsObjs.Close())
		return errors.Join(fmt.Errorf("open TLS ringbuf: %w", err), closeErr)
	}

	if id, err := fileIDOf("/proc/self/exe"); err == nil {
		c.agentFile = &id
	} else {
		log.Error().Err(err).Msg("Failed to stat agent executable. Go TLS probes may include the agent")
	}

	c.tlsInodes = make(map[fileID][]io.Closer)
	c.tlsSkip = make(map[fileID]struct{})
	c.tlsSeen = make(map[pidKey]struct{})
	c.tlsRetry = make(map[pidKey]struct{})
	c.tlsLinks = links
	c.tlsEvents = events
	c.tlsEnabled = true
	log.Info().Msg("eBPF TLS plaintext collector loaded")
	return nil
}

func (c *Collector) Start(ctx context.Context) {
	if c.tcpEnabled {
		go c.runTCP(ctx)
	}
	if c.udpEnabled {
		go c.runUDP(ctx)
		go c.runUDPIdleGC(ctx)
	}
	if c.dnsEnabled {
		go c.runDNS(ctx)
	}
	if c.streamEnabled {
		go c.runStream(ctx)
	}
	if c.tlsEnabled {
		go c.runTLS(ctx)
		go c.watchTLS(ctx)
	}
}

func (c *Collector) Close() error {
	var err error
	if c.tcpEnabled {
		if c.tcpEvents != nil {
			err = errors.Join(err, c.tcpEvents.Close())
		}
		if c.tcpLink != nil {
			err = errors.Join(err, c.tcpLink.Close())
		}
		err = errors.Join(err, c.tcpObjs.Close())
	}
	if c.udpEnabled {
		if c.udpEvents != nil {
			err = errors.Join(err, c.udpEvents.Close())
		}
		err = errors.Join(err, closeLinks(c.udpLinks))
		err = errors.Join(err, c.udpObjs.Close())
	}
	if c.dnsEnabled {
		if c.dnsEvents != nil {
			err = errors.Join(err, c.dnsEvents.Close())
		}
		err = errors.Join(err, closeLinks(c.dnsLinks))
		err = errors.Join(err, c.dnsObjs.Close())
	}
	if c.streamEnabled {
		if c.streamEvents != nil {
			err = errors.Join(err, c.streamEvents.Close())
		}
		err = errors.Join(err, closeLinks(c.streamLinks))
		err = errors.Join(err, c.streamObjs.Close())
	}
	if c.tlsEnabled {
		err = errors.Join(err, c.closeTLS())
	}
	return err
}

func attachTracing(links *[]io.Closer, prog *ebpf.Program, name string) error {
	lnk, err := link.AttachTracing(link.TracingOptions{Program: prog})
	if err != nil {
		return fmt.Errorf("attach %s: %w", name, err)
	}
	*links = append(*links, lnk)
	return nil
}

func closeLinks(links []io.Closer) error {
	var err error
	for _, lnk := range links {
		if lnk == nil {
			continue
		}
		err = errors.Join(err, lnk.Close())
	}
	return err
}
