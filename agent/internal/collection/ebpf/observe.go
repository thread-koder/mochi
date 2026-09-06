package ebpf

import (
	"net/netip"

	"github.com/thread_koder/mochi/agent/internal/collection/aggregate"
	"github.com/thread_koder/mochi/agent/internal/collection/conntrack"
	"github.com/thread_koder/mochi/agent/internal/collection/dns"
	"github.com/thread_koder/mochi/agent/internal/collection/identity"
	"github.com/thread_koder/mochi/agent/internal/metrics"
)

type flowEventKind int

const (
	flowOpen flowEventKind = iota
	flowClose
)

func (c *Collector) observeFlow(
	protocol string,
	pid uint32,
	cgroupID uint64,
	src, dst netip.Addr,
	sport, dport uint16,
	tx, rx float64,
	kind flowEventKind,
) {
	if pid == 0 && cgroupID == 0 {
		return
	}
	pod, ok := c.resolver.Resolve(pid, cgroupID)
	if !ok {
		return
	}
	if !src.IsValid() || src.IsUnspecified() || !dst.IsValid() || dst.IsUnspecified() {
		return
	}
	if dst.Unmap().IsLoopback() {
		return
	}
	if protocol == metrics.ProtocolUDP && c.serverPorts.IsBound(pod.UID, sport) {
		return
	}

	ipProto := conntrack.IPProtocol(protocol)
	actualAddr, actualPort := c.conntrackClient.ActualDst(
		ipProto,
		conntrack.Endpoint{Addr: src, Port: sport},
		conntrack.Endpoint{Addr: dst, Port: dport},
	)

	key := metrics.NewSeriesKey(
		pod.UID,
		pod.Namespace,
		pod.Name,
		protocol,
		identity.AddrKey(dst),
		int(dport),
		identity.AddrKey(actualAddr),
		int(actualPort),
	)
	dns.StampDest(&key, c.resolver, c.dnsCache, actualAddr)

	flow := aggregate.NewFlow(src, dst, sport, dport, protocol)
	switch kind {
	case flowOpen:
		c.store.ObserveConnect(flow, key, tx, rx)
	case flowClose:
		c.store.ObserveClose(flow, key, tx, rx)
	}
}
