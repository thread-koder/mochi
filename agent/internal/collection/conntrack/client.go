package conntrack

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mdlayher/netlink"
	"github.com/thread_koder/mochi/agent/internal/logger"
	"github.com/ti-mo/conntrack"
	"golang.org/x/sys/unix"
)

const (
	eventReadBuffer     = 8 << 20 // 8MB
	eventChannelSize    = 8192    // 8KB
	eventListenWorkers  = 1       // ti-mo workers share one Conn.Receive, so more only contend
	eventsGCInterval    = 1 * time.Minute
	eventsOffGCInterval = 5 * time.Second
	listenRetryInterval = 1 * time.Second
	eventsSysctlPath    = "/proc/sys/net/netfilter/nf_conntrack_events"
)

// Endpoint is one side of an L4 4-tuple used as an ActualDst argument.
type Endpoint struct {
	Addr netip.Addr
	Port uint16
}

type flowKey struct {
	proto   uint8
	srcIP   netip.Addr
	dstIP   netip.Addr
	srcPort uint16
	dstPort uint16
}

type flowValue struct {
	actualAddr netip.Addr
	actualPort uint16
	mapped     bool
	generation uint64
}

// Client resolves ClusterIP destinations via ctnetlink events and dump GC.
// Unseen 4-tuples at freeze time use a one-shot unicast lookup (not ti-mo Get).
type Client struct {
	cacheMu    sync.RWMutex
	cache      map[flowKey]flowValue
	generation uint64

	// socketMu guards the three Conn pointers only. Never hold it across netlink I/O.
	socketMu sync.Mutex
	dump     *conntrack.Conn
	lookup   *netlink.Conn
	events   *conntrack.Conn

	lookupMu sync.Mutex

	closing   chan struct{}
	serveDone chan struct{}
	closeOnce sync.Once
	startOnce sync.Once
}

func NewClient() (*Client, error) {
	dump, err := conntrack.Dial(&netlink.Config{})
	if err != nil {
		return nil, fmt.Errorf("dial conntrack dump: %w", err)
	}
	lookup, err := netlink.Dial(unix.NETLINK_NETFILTER, &netlink.Config{})
	if err != nil {
		return nil, errors.Join(fmt.Errorf("dial conntrack lookup: %w", err), dump.Close())
	}
	return &Client{
		dump:      dump,
		lookup:    lookup,
		cache:     make(map[flowKey]flowValue),
		closing:   make(chan struct{}),
		serveDone: make(chan struct{}),
	}, nil
}

func (c *Client) Close() error {
	var closeErr error
	c.closeOnce.Do(func() {
		close(c.closing)
		c.startOnce.Do(func() {
			close(c.serveDone)
		})
		<-c.serveDone

		c.socketMu.Lock()
		dump := c.dump
		events := c.events
		c.dump = nil
		c.events = nil
		c.socketMu.Unlock()

		if events != nil {
			if err := events.Close(); err != nil {
				closeErr = errors.Join(closeErr, fmt.Errorf("close conntrack events: %w", err))
			}
		}
		if dump != nil {
			if err := dump.Close(); err != nil {
				closeErr = errors.Join(closeErr, fmt.Errorf("close conntrack dump: %w", err))
			}
		}

		c.lookupMu.Lock()
		c.socketMu.Lock()
		lookup := c.lookup
		c.lookup = nil
		c.socketMu.Unlock()
		if lookup != nil {
			if err := lookup.Close(); err != nil {
				closeErr = errors.Join(closeErr, fmt.Errorf("close conntrack lookup: %w", err))
			}
		}
		c.lookupMu.Unlock()
	})
	return closeErr
}

// Start joins ctnetlink when events are enabled, runs the first dump GC, and
// keeps the serve loop running. Blocks until that first resync finishes.
func (c *Client) Start(ctx context.Context) error {
	log := logger.WithComponent("conntrack")
	ready := make(chan struct{})
	events := eventsEnabled()

	c.startOnce.Do(func() {
		if events {
			go c.serveEvents(ctx, ready)
			return
		}
		go c.serveDumpOnly(ctx, ready)
	})

	select {
	case <-ctx.Done():
		<-ready
		return context.Cause(ctx)
	case <-ready:
		log.Info().Bool("events", events).Msg("Conntrack started")
		return nil
	}
}

// Missing sysctl means the host is events-capable (path exists on kernels with nf_conntrack).
func eventsEnabled() bool {
	raw, err := os.ReadFile(eventsSysctlPath)
	if err != nil {
		return true
	}
	value := strings.TrimSpace(string(raw))
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return true
	}
	return parsed != 0
}

func (c *Client) applyDumpGC(startGeneration uint64, flows []conntrack.Flow) {
	seen := make(map[flowKey]struct{}, len(flows))
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	for i := range flows {
		key, value, ok := mappingFromFlow(&flows[i], startGeneration)
		if !ok {
			continue
		}
		seen[key] = struct{}{}
		c.cache[key] = value
	}
	// Pre-listen flows never emit DESTROY. Evict stale mapped keys and negatives
	// stamped at or before this dump. Keep generation > startGeneration (in-flight
	// NEW/lookup during the dump).
	for key, value := range c.cache {
		if value.generation > startGeneration {
			continue
		}
		if value.mapped {
			if _, ok := seen[key]; ok {
				continue
			}
		}
		delete(c.cache, key)
	}
}

func (c *Client) applyEvent(ev conntrack.Event) {
	if ev.Flow == nil {
		return
	}
	switch ev.Type {
	case conntrack.EventNew, conntrack.EventUpdate:
		c.upsertFlow(ev.Flow)
	case conntrack.EventDestroy:
		key, ok := keyFromOrig(ev.Flow)
		if !ok {
			return
		}
		c.cacheMu.Lock()
		delete(c.cache, key)
		c.cacheMu.Unlock()
	}
}

func (c *Client) upsertFlow(flow *conntrack.Flow) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	key, value, ok := mappingFromFlow(flow, c.generation)
	if !ok {
		return
	}
	c.cache[key] = value
}

func mappingFromFlow(flow *conntrack.Flow, generation uint64) (flowKey, flowValue, bool) {
	key, ok := keyFromOrig(flow)
	if !ok {
		return flowKey{}, flowValue{}, false
	}
	actualAddr := flow.TupleReply.IP.SourceAddress.Unmap()
	actualPort := flow.TupleReply.Proto.SourcePort
	if !actualAddr.IsValid() || actualPort == 0 || actualAddr.IsLoopback() {
		return key, flowValue{mapped: false, generation: generation}, true
	}
	return key, flowValue{
		actualAddr: actualAddr,
		actualPort: actualPort,
		mapped:     true,
		generation: generation,
	}, true
}

func keyFromOrig(flow *conntrack.Flow) (flowKey, bool) {
	proto := flow.TupleOrig.Proto.Protocol
	if proto != unix.IPPROTO_TCP && proto != unix.IPPROTO_UDP {
		return flowKey{}, false
	}
	src := flow.TupleOrig.IP.SourceAddress.Unmap()
	dst := flow.TupleOrig.IP.DestinationAddress.Unmap()
	if !src.IsValid() || !dst.IsValid() {
		return flowKey{}, false
	}
	return flowKey{
		proto:   proto,
		srcIP:   src,
		dstIP:   dst,
		srcPort: flow.TupleOrig.Proto.SourcePort,
		dstPort: flow.TupleOrig.Proto.DestinationPort,
	}, true
}

func (c *Client) isClosing() bool {
	select {
	case <-c.closing:
		return true
	default:
		return false
	}
}

// IPProtocol maps a Prometheus protocol label to the IP protocol number.
func IPProtocol(protocol string) uint8 {
	switch protocol {
	case "tcp":
		return unix.IPPROTO_TCP
	case "udp":
		return unix.IPPROTO_UDP
	default:
		return 0
	}
}

// ActualDst returns the post-NAT destination, or the original dest when lookup
// misses or the mapped peer is loopback.
func (c *Client) ActualDst(proto uint8, src, dst Endpoint) (netip.Addr, uint16) {
	if c == nil {
		return dst.Addr, dst.Port
	}
	key := flowKey{
		proto:   proto,
		srcIP:   src.Addr.Unmap(),
		dstIP:   dst.Addr.Unmap(),
		srcPort: src.Port,
		dstPort: dst.Port,
	}

	c.cacheMu.RLock()
	value, ok := c.cache[key]
	c.cacheMu.RUnlock()
	if ok {
		if !value.mapped {
			return dst.Addr, dst.Port
		}
		return value.actualAddr, value.actualPort
	}

	flow, found, err := c.lookupFlow(proto, src, dst)
	if err != nil {
		log := logger.WithComponent("conntrack")
		log.Error().Err(err).Msg("Conntrack lookup failed")
		return dst.Addr, dst.Port
	}

	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if existing, ok := c.cache[key]; ok {
		if !existing.mapped {
			return dst.Addr, dst.Port
		}
		return existing.actualAddr, existing.actualPort
	}
	if !found {
		c.cache[key] = flowValue{mapped: false, generation: c.generation}
		return dst.Addr, dst.Port
	}

	_, value, ok = mappingFromFlow(&flow, c.generation)
	if !ok {
		c.cache[key] = flowValue{mapped: false, generation: c.generation}
		return dst.Addr, dst.Port
	}
	c.cache[key] = value
	if !value.mapped {
		return dst.Addr, dst.Port
	}
	return value.actualAddr, value.actualPort
}
