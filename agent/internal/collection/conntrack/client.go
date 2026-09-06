package conntrack

import (
	"context"
	"fmt"
	"net/netip"
	"sync"
	"time"

	"github.com/mdlayher/netlink"
	"github.com/thread_koder/mochi/agent/internal/logger"
	"github.com/ti-mo/conntrack"
)

const (
	ProtocolTCP uint8 = 6
	ProtocolUDP uint8 = 17
)

// Endpoint is one side of an L4 4-tuple.
type Endpoint struct {
	Addr netip.Addr
	Port uint16
}

type flowKey struct {
	proto   uint8
	srcIP   string
	dstIP   string
	srcPort uint16
	dstPort uint16
}

type flowValue struct {
	actualAddr netip.Addr
	actualPort uint16
}

// Client resolves ClusterIP destinations via a periodically refreshed conntrack dump.
type Client struct {
	mu    sync.RWMutex
	conn  *conntrack.Conn
	cache map[flowKey]flowValue
}

func NewClient() (*Client, error) {
	conn, err := conntrack.Dial(&netlink.Config{})
	if err != nil {
		return nil, fmt.Errorf("dial conntrack: %w", err)
	}
	return &Client{
		conn:  conn,
		cache: make(map[flowKey]flowValue),
	}, nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("close conntrack: %w", err)
	}
	c.conn = nil
	return nil
}

func (c *Client) StartRefresh(ctx context.Context, interval time.Duration) {
	log := logger.WithComponent("conntrack")
	if err := c.refresh(); err != nil {
		log.Error().Err(err).Msg("Initial conntrack refresh failed")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.refresh(); err != nil {
				log.Error().Err(err).Msg("Conntrack refresh failed")
			}
		}
	}
}

func (c *Client) refresh() error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("conntrack client closed")
	}

	flows, err := conn.Dump(nil)
	if err != nil {
		return fmt.Errorf("dump conntrack: %w", err)
	}

	next := make(map[flowKey]flowValue, len(flows))
	for _, flow := range flows {
		proto := flow.TupleOrig.Proto.Protocol
		if proto != ProtocolTCP && proto != ProtocolUDP {
			continue
		}
		origSrc := flow.TupleOrig.IP.SourceAddress
		origDst := flow.TupleOrig.IP.DestinationAddress
		if !origSrc.IsValid() || !origDst.IsValid() {
			continue
		}
		actualAddr := flow.TupleReply.IP.SourceAddress
		actualPort := flow.TupleReply.Proto.SourcePort
		if !actualAddr.IsValid() || actualPort == 0 {
			continue
		}
		key := flowKey{
			proto:   proto,
			srcIP:   origSrc.Unmap().String(),
			dstIP:   origDst.Unmap().String(),
			srcPort: flow.TupleOrig.Proto.SourcePort,
			dstPort: flow.TupleOrig.Proto.DestinationPort,
		}
		next[key] = flowValue{
			actualAddr: actualAddr.Unmap(),
			actualPort: actualPort,
		}
	}

	c.mu.Lock()
	c.cache = next
	c.mu.Unlock()
	return nil
}

// IPProtocol maps a Prometheus protocol label to the IP protocol number.
func IPProtocol(protocol string) uint8 {
	switch protocol {
	case "tcp":
		return ProtocolTCP
	case "udp":
		return ProtocolUDP
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
		srcIP:   src.Addr.Unmap().String(),
		dstIP:   dst.Addr.Unmap().String(),
		srcPort: src.Port,
		dstPort: dst.Port,
	}
	c.mu.RLock()
	value, ok := c.cache[key]
	c.mu.RUnlock()
	if !ok || value.actualAddr.Unmap().IsLoopback() {
		return dst.Addr, dst.Port
	}
	return value.actualAddr, value.actualPort
}
