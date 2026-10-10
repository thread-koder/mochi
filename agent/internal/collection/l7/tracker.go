package l7

import (
	"context"
	"net/netip"
	"sync"
	"time"

	"github.com/thread_koder/mochi/agent/internal/collection/aggregate"
	"github.com/thread_koder/mochi/agent/internal/collection/conntrack"
	"github.com/thread_koder/mochi/agent/internal/collection/dns"
	"github.com/thread_koder/mochi/agent/internal/collection/http1"
	"github.com/thread_koder/mochi/agent/internal/collection/http2"
	"github.com/thread_koder/mochi/agent/internal/collection/identity"
	"github.com/thread_koder/mochi/agent/internal/metrics"
	"github.com/thread_koder/mochi/agent/internal/span"
)

const (
	maxOutstanding = 32768
	connIdle       = 60 * time.Second
	gcInterval     = 10 * time.Second
)

type protoKind uint8

const (
	protoUnknown protoKind = iota
	protoHTTP1
	protoHTTP2
)

type completedHop struct {
	key          metrics.SeriesKey
	method       string
	route        string
	statusClass  string
	statusCode   int
	grpc         bool
	grpcStatus   *int
	start        time.Time
	end          time.Time
	traceID      [span.TraceIDSize]byte
	spanID       [span.SpanIDSize]byte
	parentSpanID [span.SpanIDSize]byte
	sampled      bool
}

// pendingRequest is a hop that missed Store.Lookup and still needs identity.
type pendingRequest struct {
	flow         aggregate.Flow
	pid          uint32
	cgroupID     uint64
	src          netip.Addr
	dst          netip.Addr
	sport        uint16
	dport        uint16
	kind         uint8
	method       string
	path         string
	grpc         bool
	stream       uint32
	traceparent  string
	grpcTraceBin string
	now          time.Time
}

func (p pendingRequest) fromTLS() bool {
	return p.kind == KindOpenSSL || p.kind == KindGoTLS
}

type connState struct {
	opaque    bool
	preferTLS bool
	lastSeen  time.Time
	proto     protoKind
	h1        *http1.Conn
	h2        *http2.Conn
}

// Tracker demuxes HTTP/1 and HTTP/2 prefixes into hop RED and sampled spans.
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
	exporter    *span.Exporter
	sampleRate  float64
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
	exporter *span.Exporter,
	sampleRate float64,
	maxSeries int,
) *Tracker {
	return &Tracker{
		conns:      make(map[aggregate.Flow]*connState),
		router:     NewRouter(),
		registry:   registry,
		store:      store,
		resolver:   resolver,
		conntrack:  conntrackClient,
		dnsCache:   dnsCache,
		exporter:   exporter,
		sampleRate: sampleRate,
		httpMax:    maxSeries,
		series:     make(map[metrics.HTTPSeriesKey]struct{}),
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
	var pending []pendingRequest

	t.mu.Lock()
	conn := t.conns[flow]
	if conn == nil {
		kind := classify(chunk.Data)
		if kind == protoUnknown {
			t.mu.Unlock()
			return
		}
		conn = &connState{proto: kind, lastSeen: now}
		if kind == protoHTTP1 {
			conn.h1 = &http1.Conn{}
		} else {
			conn.h2 = http2.NewConn()
		}
		t.conns[flow] = conn
	}
	conn.lastSeen = now

	if conn.opaque {
		t.mu.Unlock()
		return
	}

	if chunk.fromTLS() {
		conn.preferTLS = true
	} else if conn.preferTLS && chunk.Kind == KindSocket {
		t.mu.Unlock()
		return
	}

	switch conn.proto {
	case protoHTTP1:
		hops, pending = t.feedHTTP1(conn, flow, chunk, now)
	case protoHTTP2:
		hops, pending = t.feedHTTP2(conn, flow, chunk, now)
	}
	t.mu.Unlock()

	for _, p := range pending {
		key, ok := t.liveIdentity(p)
		if !ok {
			continue
		}
		t.mu.Lock()
		conn := t.conns[p.flow]
		if conn == nil || conn.opaque {
			t.mu.Unlock()
			continue
		}
		if p.kind == KindSocket && conn.preferTLS {
			t.mu.Unlock()
			continue
		}
		t.enqueuePending(conn, p, key)
		t.mu.Unlock()
	}

	for _, hop := range hops {
		t.recordHop(hop)
	}
}

func classify(data []byte) protoKind {
	if http2.LooksLikePreface(data) {
		return protoHTTP2
	}
	if http2.LooksLikeFrames(data) {
		return protoHTTP2
	}
	if http1.LooksLikeStart(data) {
		return protoHTTP1
	}
	return protoUnknown
}

func (t *Tracker) feedHTTP1(conn *connState, flow aggregate.Flow, chunk Chunk, now time.Time) ([]completedHop, []pendingRequest) {
	reqs, completions, opaque := conn.h1.Feed(chunk.Dir, chunk.Data, now, &t.outstanding)
	replaceTLS := chunk.fromTLS()

	var pending []pendingRequest
	for _, req := range reqs {
		if key, ok := t.store.Lookup(flow); ok {
			req.Route = t.routeFor(key, req.Path, false)
			req.Key = key
			t.enqueueHTTP1(conn, req, replaceTLS)
			continue
		}
		pending = append(pending, pendingFrom(flow, chunk, req.Method, req.Path, false, 0, req.Traceparent, "", now))
	}

	hops := make([]completedHop, 0, len(completions))
	for _, hop := range completions {
		hops = append(hops, completedHop{
			key:          hop.Key,
			method:       hop.Method,
			route:        hop.Route,
			statusClass:  hop.StatusClass,
			statusCode:   hop.StatusCode,
			start:        hop.Start,
			end:          hop.End,
			traceID:      hop.TraceID,
			spanID:       hop.SpanID,
			parentSpanID: hop.ParentSpanID,
			sampled:      hop.Sampled,
		})
	}

	if opaque {
		t.outstanding -= conn.h1.DropOutstanding()
		conn.opaque = true
	}
	return hops, pending
}

func (t *Tracker) feedHTTP2(conn *connState, flow aggregate.Flow, chunk Chunk, now time.Time) ([]completedHop, []pendingRequest) {
	reqs, completions, stop := conn.h2.Feed(chunk.Dir, chunk.Data, chunk.truncated(), now, &t.outstanding)
	replaceTLS := chunk.fromTLS()

	var pending []pendingRequest
	for _, req := range reqs {
		if key, ok := t.store.Lookup(flow); ok {
			req.Route = t.routeFor(key, req.Path, req.GRPC)
			req.Key = key
			t.enqueueHTTP2(conn, req, replaceTLS)
			continue
		}
		pending = append(pending, pendingFrom(flow, chunk, req.Method, req.Path, req.GRPC, req.StreamID, req.Traceparent, req.GRPCTraceBin, now))
	}

	hops := make([]completedHop, 0, len(completions))
	for _, hop := range completions {
		hops = append(hops, completedHop{
			key:          hop.Key,
			method:       hop.Method,
			route:        hop.Route,
			statusClass:  hop.StatusClass,
			statusCode:   hop.StatusCode,
			grpc:         hop.GRPC,
			grpcStatus:   hop.GRPCStatus,
			start:        hop.Start,
			end:          hop.End,
			traceID:      hop.TraceID,
			spanID:       hop.SpanID,
			parentSpanID: hop.ParentSpanID,
			sampled:      hop.Sampled,
		})
	}

	if stop {
		conn.opaque = true
	}
	return hops, pending
}

func pendingFrom(flow aggregate.Flow, chunk Chunk, method, path string, grpc bool, stream uint32, traceparent, grpcTraceBin string, now time.Time) pendingRequest {
	return pendingRequest{
		flow:         flow,
		pid:          chunk.Pid,
		cgroupID:     chunk.CgroupID,
		src:          chunk.Src,
		dst:          chunk.Dst,
		sport:        chunk.Sport,
		dport:        chunk.Dport,
		kind:         chunk.Kind,
		method:       method,
		path:         path,
		grpc:         grpc,
		stream:       stream,
		traceparent:  traceparent,
		grpcTraceBin: grpcTraceBin,
		now:          now,
	}
}

func (t *Tracker) routeFor(key metrics.SeriesKey, path string, grpc bool) string {
	if grpc {
		return CapRoute(path)
	}
	return t.router.Template(DestKey(key.DstPodUID, key.ActualDstIP, key.ActualDstPort), path)
}

func (t *Tracker) enqueuePending(conn *connState, pending pendingRequest, key metrics.SeriesKey) {
	replaceTLS := pending.fromTLS()
	route := t.routeFor(key, pending.path, pending.grpc)
	switch conn.proto {
	case protoHTTP1:
		t.enqueueHTTP1(conn, http1.Request{
			Method:      pending.method,
			Path:        pending.path,
			Route:       route,
			Key:         key,
			Start:       pending.now,
			Traceparent: pending.traceparent,
		}, replaceTLS)
	case protoHTTP2:
		t.enqueueHTTP2(conn, http2.Request{
			StreamID:     pending.stream,
			Method:       pending.method,
			Path:         pending.path,
			Route:        route,
			GRPC:         pending.grpc,
			Key:          key,
			Start:        pending.now,
			Traceparent:  pending.traceparent,
			GRPCTraceBin: pending.grpcTraceBin,
		}, replaceTLS)
	}
}

func (t *Tracker) enqueueHTTP1(conn *connState, req http1.Request, replaceTLS bool) {
	ids := t.stampIDs(req.Traceparent, "")
	req.TraceID = ids.traceID
	req.SpanID = ids.spanID
	req.ParentSpanID = ids.parentSpanID
	req.Sampled = ids.sampled
	conn.h1.Enqueue(req, replaceTLS, &t.outstanding, maxOutstanding)
}

func (t *Tracker) enqueueHTTP2(conn *connState, req http2.Request, replaceTLS bool) {
	ids := t.stampIDs(req.Traceparent, req.GRPCTraceBin)
	req.TraceID = ids.traceID
	req.SpanID = ids.spanID
	req.ParentSpanID = ids.parentSpanID
	req.Sampled = ids.sampled
	conn.h2.Enqueue(req, replaceTLS, &t.outstanding, maxOutstanding)
}

type hopIDs struct {
	traceID      [span.TraceIDSize]byte
	spanID       [span.SpanIDSize]byte
	parentSpanID [span.SpanIDSize]byte
	sampled      bool
}

// stampIDs prefers W3C traceparent over grpc-trace-bin. On success: adopt
// trace_id + parent, mint a new span_id. W3C sampled is honored as-is. OC
// deferred (or options absent) uses TraceIDRatioBased on the adopted id.
func (t *Tracker) stampIDs(traceparent, grpcTraceBin string) hopIDs {
	var tc span.TraceContext
	ok := false
	if traceparent != "" {
		tc, ok = span.ParseTraceparent(traceparent)
	}
	if !ok && grpcTraceBin != "" {
		tc, ok = span.ParseGRPCTraceBin(grpcTraceBin)
	}
	if ok {
		sid, err := span.NewSpanID()
		if err != nil {
			return hopIDs{}
		}
		ids := hopIDs{
			traceID:      tc.TraceID,
			spanID:       sid,
			parentSpanID: tc.ParentSpanID,
		}
		if tc.Deferred {
			ids.sampled = span.Sampled(tc.TraceID, t.sampleRate)
		} else {
			ids.sampled = tc.Sampled
		}
		return ids
	}

	tid, sid, err := span.NewIDs()
	if err != nil {
		return hopIDs{}
	}
	return hopIDs{
		traceID: tid,
		spanID:  sid,
		sampled: span.Sampled(tid, t.sampleRate),
	}
}

func (t *Tracker) liveIdentity(pending pendingRequest) (metrics.SeriesKey, bool) {
	if pending.pid == 0 && pending.cgroupID == 0 {
		return metrics.SeriesKey{}, false
	}
	pod, ok := t.resolver.Resolve(pending.pid, pending.cgroupID)
	if !ok {
		return metrics.SeriesKey{}, false
	}
	actualAddr, actualPort := t.conntrack.ActualDst(
		conntrack.IPProtocol(metrics.ProtocolTCP),
		conntrack.Endpoint{Addr: pending.src, Port: pending.sport},
		conntrack.Endpoint{Addr: pending.dst, Port: pending.dport},
	)
	key := metrics.NewSeriesKey(
		pod.UID,
		pod.Namespace,
		pod.Name,
		metrics.ProtocolTCP,
		identity.AddrKey(pending.dst),
		int(pending.dport),
		identity.AddrKey(actualAddr),
		int(actualPort),
	)
	dns.StampDest(&key, t.resolver, t.dnsCache, actualAddr)
	return key, true
}

func (t *Tracker) recordHop(hop completedHop) {
	t.offerSpan(hop)
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
	t.registry.RecordHTTP(hop.key, hop.method, hop.route, hop.statusClass, hop.end.Sub(hop.start).Seconds())
}

func (t *Tracker) offerSpan(hop completedHop) {
	if t.exporter == nil {
		return
	}
	if span.IsZeroTraceID(hop.traceID) || span.IsZeroSpanID(hop.spanID) {
		return
	}

	keep := hop.sampled
	if hop.grpc {
		if hop.grpcStatus != nil && *hop.grpcStatus != 0 {
			keep = true
		}
	} else if hop.statusClass == "5xx" {
		keep = true
	}
	if !keep {
		return
	}

	sampledBy := span.SampledByHead
	if !hop.sampled {
		sampledBy = span.SampledByError
	}

	t.exporter.Offer(span.Hop{
		TraceID:      hop.traceID,
		SpanID:       hop.spanID,
		ParentSpanID: hop.parentSpanID,
		Start:        hop.start,
		End:          hop.end,
		Method:       hop.method,
		Route:        hop.route,
		StatusClass:  hop.statusClass,
		StatusCode:   hop.statusCode,
		GRPC:         hop.grpc,
		GRPCStatus:   hop.grpcStatus,
		SampledBy:    sampledBy,
		Key:          hop.key,
	})
}

func (t *Tracker) gc(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for flow, conn := range t.conns {
		idle := false
		switch conn.proto {
		case protoHTTP1:
			t.outstanding -= conn.h1.GCExpired(now, connIdle)
			idle = conn.h1.Idle()
		case protoHTTP2:
			t.outstanding -= conn.h2.GCExpired(now, connIdle)
			idle = conn.h2.Idle()
		default:
			idle = true
		}
		if now.Sub(conn.lastSeen) > connIdle && idle {
			delete(t.conns, flow)
		}
	}
}
