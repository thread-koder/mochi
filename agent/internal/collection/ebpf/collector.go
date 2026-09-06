package ebpf

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/thread_koder/mochi/agent/internal/collection/aggregate"
	"github.com/thread_koder/mochi/agent/internal/collection/conntrack"
	"github.com/thread_koder/mochi/agent/internal/collection/dns"
	"github.com/thread_koder/mochi/agent/internal/collection/identity"
	"github.com/thread_koder/mochi/agent/internal/logger"
)

// ServerPorts marks local UDP ports where sendmsg is server-side (bound/unconnected).
// Only the UDP fexit hook consults this. TCP state transitions are client-shaped.
type ServerPorts interface {
	IsBound(podUID string, port uint16) bool
}

// Collector owns loaded eBPF programs and event loops for TCP, UDP, and DNS.
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

	store           *aggregate.Store
	resolver        *identity.Resolver
	conntrackClient *conntrack.Client
	serverPorts     ServerPorts
	dnsCache        *dns.Cache
}

func Load(
	store *aggregate.Store,
	resolver *identity.Resolver,
	conntrackClient *conntrack.Client,
	serverPorts ServerPorts,
	dnsCache *dns.Cache,
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
	attach := func(prog *ebpf.Program) error {
		lnk, err := link.AttachTracing(link.TracingOptions{
			Program: prog,
		})
		if err != nil {
			return err
		}
		links = append(links, lnk)
		return nil
	}

	if err := attach(c.udpObjs.MochiUdpSendmsg); err != nil {
		closeErr := errors.Join(closeLinks(links), c.udpObjs.Close())
		return errors.Join(fmt.Errorf("attach fexit udp_sendmsg: %w", err), closeErr)
	}
	if err := attach(c.udpObjs.MochiUdpv6Sendmsg); err != nil {
		closeErr := errors.Join(closeLinks(links), c.udpObjs.Close())
		return errors.Join(fmt.Errorf("attach fexit udpv6_sendmsg: %w", err), closeErr)
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
	attach := func(prog *ebpf.Program, name string) error {
		lnk, err := link.AttachTracing(link.TracingOptions{
			Program: prog,
		})
		if err != nil {
			return fmt.Errorf("attach %s: %w", name, err)
		}
		links = append(links, lnk)
		return nil
	}

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
		if err := attach(step.prog, step.name); err != nil {
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
	return err
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
