package http1

import (
	"bytes"
	"context"
	"net/netip"
	"sync"
	"time"

	"github.com/thread_koder/mochi/agent/internal/collection/aggregate"
	"github.com/thread_koder/mochi/agent/internal/collection/conntrack"
	"github.com/thread_koder/mochi/agent/internal/collection/dns"
	"github.com/thread_koder/mochi/agent/internal/collection/identity"
	"github.com/thread_koder/mochi/agent/internal/metrics"
)

const (
	maxOutstanding = 32768
	connIdle       = 60 * time.Second
	gcInterval     = 10 * time.Second
)

// Chunk is one stream/TLS ringbuf payload for the HTTP/1 tracker.
type Chunk struct {
	Pid      uint32
	CgroupID uint64
	Family   uint16
	Sport    uint16
	Dport    uint16
	Dir      uint8
	Kind     uint8
	Src      netip.Addr
	Dst      netip.Addr
	Data     []byte
}

type pendingReq struct {
	method string
	route  string
	key    metrics.SeriesKey
	start  time.Time
}

type connState struct {
	opaque      bool
	preferTLS   bool
	lastSeen    time.Time
	sendBuf     []byte
	recvBuf     []byte
	outstanding []pendingReq
}

type completedHop struct {
	key         metrics.SeriesKey
	method      string
	route       string
	statusClass string
	seconds     float64
}

type liveNeed struct {
	flow  aggregate.Flow
	chunk Chunk
	msg   Message
	now   time.Time
}

// Tracker pairs client-role HTTP/1 request/response prefixes into hop RED.
// Owns the HTTP MAX_SERIES budget (drop new label sets, never delete).
type Tracker struct {
	mu          sync.Mutex
	conns       map[aggregate.Flow]*connState
	router      *Router
	registry    *metrics.Registry
	store       *aggregate.Store
	resolver    *identity.Resolver
	conntrack   *conntrack.Client
	dnsCache    *dns.Cache
	outstanding int

	seriesMu sync.Mutex
	httpMax  int
	series   map[metrics.HTTPSeriesKey]struct{}
}

func NewTracker(
	registry *metrics.Registry,
	store *aggregate.Store,
	resolver *identity.Resolver,
	conntrackClient *conntrack.Client,
	dnsCache *dns.Cache,
	maxSeries int,
) *Tracker {
	return &Tracker{
		conns:     make(map[aggregate.Flow]*connState),
		router:    NewRouter(),
		registry:  registry,
		store:     store,
		resolver:  resolver,
		conntrack: conntrackClient,
		dnsCache:  dnsCache,
		httpMax:   maxSeries,
		series:    make(map[metrics.HTTPSeriesKey]struct{}),
	}
}

func (t *Tracker) Start(ctx context.Context) {
	ticker := time.NewTicker(gcInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.gc(time.Now())
		}
	}
}

func (t *Tracker) Handle(chunk Chunk) {
	if !chunk.Src.IsValid() || chunk.Src.IsUnspecified() || !chunk.Dst.IsValid() || chunk.Dst.IsUnspecified() {
		return
	}
	if chunk.Dst.Unmap().IsLoopback() {
		return
	}
	if len(chunk.Data) == 0 {
		return
	}

	flow := aggregate.NewFlow(chunk.Src, chunk.Dst, chunk.Sport, chunk.Dport, metrics.ProtocolTCP)
	now := time.Now()

	var hops []completedHop
	var needs []liveNeed

	t.mu.Lock()
	conn := t.conns[flow]
	if conn == nil {
		if !looksLikeHTTPStart(chunk.Data) {
			t.mu.Unlock()
			return
		}
		conn = &connState{}
		t.conns[flow] = conn
	}
	conn.lastSeen = now

	if conn.opaque {
		t.mu.Unlock()
		return
	}

	if chunk.Kind == KindOpenSSL || chunk.Kind == KindGoTLS {
		conn.preferTLS = true
	} else if conn.preferTLS && chunk.Kind == KindSocket {
		t.mu.Unlock()
		return
	}

	data := chunk.Data
	var buf *[]byte
	switch chunk.Dir {
	case DirSend:
		buf = &conn.sendBuf
	case DirRecv:
		buf = &conn.recvBuf
	default:
		t.mu.Unlock()
		return
	}
	if len(*buf) == 0 && !looksLikeHTTPStart(data) {
		t.mu.Unlock()
		return
	}
	if len(*buf) > 0 {
		combined := make([]byte, 0, len(*buf)+len(data))
		combined = append(combined, *buf...)
		combined = append(combined, data...)
		data = combined
		*buf = nil
	}

	msgs, leftover, opaque := ParsePrefix(data)
	if opaque {
		conn.opaque = true
		conn.sendBuf = nil
		conn.recvBuf = nil
		t.outstanding -= len(conn.outstanding)
		conn.outstanding = nil
	}
	if len(leftover) > 0 {
		if len(leftover) > incompleteCap {
			*buf = nil
		} else {
			*buf = bytes.Clone(leftover)
		}
	}

	for _, msg := range msgs {
		if msg.HTTP2 {
			conn.opaque = true
			continue
		}
		if msg.Request {
			if chunk.Dir != DirSend {
				continue
			}
			if msg.Connect {
				conn.opaque = true
				continue
			}
			if key, ok := t.store.Lookup(flow); ok {
				t.enqueueRequest(conn, chunk, msg, key, now)
			} else {
				needs = append(needs, liveNeed{flow: flow, chunk: chunk, msg: msg, now: now})
			}
			if msg.Opaque {
				conn.opaque = true
			}
			continue
		}
		if chunk.Dir != DirRecv {
			continue
		}
		if hop, ok := t.popResponse(conn, msg, now); ok {
			hops = append(hops, hop)
		}
		if msg.Opaque {
			conn.opaque = true
		}
	}

	if conn.opaque {
		conn.sendBuf = nil
		conn.recvBuf = nil
	}
	t.mu.Unlock()

	for _, need := range needs {
		key, ok := t.liveIdentity(need.chunk)
		if !ok {
			continue
		}
		t.mu.Lock()
		conn := t.conns[need.flow]
		if conn == nil || conn.opaque {
			t.mu.Unlock()
			continue
		}
		if need.chunk.Kind == KindSocket && conn.preferTLS {
			t.mu.Unlock()
			continue
		}
		t.enqueueRequest(conn, need.chunk, need.msg, key, need.now)
		t.mu.Unlock()
	}

	for _, hop := range hops {
		t.recordHop(hop)
	}
}

func (t *Tracker) enqueueRequest(conn *connState, chunk Chunk, msg Message, key metrics.SeriesKey, now time.Time) {
	route := t.router.Template(destKey(key.DstPodUID, key.ActualDstIP, key.ActualDstPort), msg.Path)
	req := pendingReq{method: msg.Method, route: route, key: key, start: now}

	if chunk.Kind == KindOpenSSL || chunk.Kind == KindGoTLS {
		if n := len(conn.outstanding); n > 0 {
			last := &conn.outstanding[n-1]
			if last.method == req.method && last.route == req.route {
				*last = req
				return
			}
		}
	}

	if t.outstanding >= maxOutstanding {
		return
	}
	conn.outstanding = append(conn.outstanding, req)
	t.outstanding++
}

func (t *Tracker) popResponse(conn *connState, msg Message, now time.Time) (completedHop, bool) {
	if len(conn.outstanding) == 0 {
		return completedHop{}, false
	}
	if msg.Status >= 100 && msg.Status < 200 && msg.Status != 101 {
		return completedHop{}, false
	}

	req := conn.outstanding[0]
	conn.outstanding = conn.outstanding[1:]
	t.outstanding--

	if msg.Status == 101 {
		return completedHop{}, false
	}
	class := statusClass(msg.Status)
	if class == "" {
		return completedHop{}, false
	}
	return completedHop{
		key:         req.key,
		method:      req.method,
		route:       req.route,
		statusClass: class,
		seconds:     now.Sub(req.start).Seconds(),
	}, true
}

func (t *Tracker) liveIdentity(chunk Chunk) (metrics.SeriesKey, bool) {
	if chunk.Pid == 0 && chunk.CgroupID == 0 {
		return metrics.SeriesKey{}, false
	}
	pod, ok := t.resolver.Resolve(chunk.Pid, chunk.CgroupID)
	if !ok {
		return metrics.SeriesKey{}, false
	}
	actualAddr, actualPort := t.conntrack.ActualDst(
		conntrack.IPProtocol(metrics.ProtocolTCP),
		conntrack.Endpoint{Addr: chunk.Src, Port: chunk.Sport},
		conntrack.Endpoint{Addr: chunk.Dst, Port: chunk.Dport},
	)
	key := metrics.NewSeriesKey(
		pod.UID,
		pod.Namespace,
		pod.Name,
		metrics.ProtocolTCP,
		identity.AddrKey(chunk.Dst),
		int(chunk.Dport),
		identity.AddrKey(actualAddr),
		int(actualPort),
	)
	dns.StampDest(&key, t.resolver, t.dnsCache, actualAddr)
	return key, true
}

func (t *Tracker) recordHop(hop completedHop) {
	httpKey := metrics.HTTPSeriesKey{SeriesKey: hop.key, Method: hop.method, Route: hop.route}
	t.seriesMu.Lock()
	_, known := t.series[httpKey]
	if !known {
		if len(t.series) >= t.httpMax {
			t.seriesMu.Unlock()
			return
		}
		t.series[httpKey] = struct{}{}
	}
	t.seriesMu.Unlock()
	t.registry.RecordHTTP(hop.key, hop.method, hop.route, hop.statusClass, hop.seconds)
}

func (t *Tracker) gc(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for flow, conn := range t.conns {
		kept := conn.outstanding[:0]
		for _, req := range conn.outstanding {
			if now.Sub(req.start) > connIdle {
				t.outstanding--
				continue
			}
			kept = append(kept, req)
		}
		conn.outstanding = kept
		if now.Sub(conn.lastSeen) > connIdle && len(conn.outstanding) == 0 {
			delete(t.conns, flow)
		}
	}
}
