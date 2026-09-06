package aggregate

import (
	"net/netip"
	"sync"
	"time"

	"github.com/thread_koder/mochi/agent/internal/metrics"
)

// Flow is a client-outbound L4 4-tuple (pre-NAT dest).
type Flow struct {
	Src, Dst         netip.Addr
	SrcPort, DstPort uint16
	Protocol         string
}

func NewFlow(src, dst netip.Addr, sport, dport uint16, protocol string) Flow {
	return Flow{
		Src:      src.Unmap(),
		Dst:      dst.Unmap(),
		SrcPort:  sport,
		DstPort:  dport,
		Protocol: protocol,
	}
}

// SeedSocket is one active client socket from a /proc snapshot.
type SeedSocket struct {
	Flow     Flow
	Fallback metrics.SeriesKey // used when the 4-tuple is not yet bound
}

type seriesStats struct {
	eventActive float64
	seedActive  float64
	lastTouch   time.Time
}

func (st *seriesStats) gauge() float64 { return st.eventActive + st.seedActive }

// Store pre-aggregates series on the node before Prometheus export.
// flows freezes each socket's SeriesKey at open so Prom labels stay stable
// (counters cannot be retagged when NAT later hits or DNS fills after a race).
type Store struct {
	mu        sync.Mutex
	maxSeries int
	series    map[metrics.SeriesKey]*seriesStats
	flows     map[Flow]metrics.SeriesKey
	registry  *metrics.Registry
}

func NewStore(registry *metrics.Registry, maxSeries int) *Store {
	return &Store{
		maxSeries: maxSeries,
		series:    make(map[metrics.SeriesKey]*seriesStats),
		flows:     make(map[Flow]metrics.SeriesKey),
		registry:  registry,
	}
}

func (s *Store) ObserveConnect(flow Flow, key metrics.SeriesKey, txDelta, rxDelta float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if oldKey, ok := s.flows[flow]; ok {
		s.dropEventActiveLocked(oldKey, now)
	}

	s.flows[flow] = key
	stats := s.getOrCreateLocked(key)
	stats.eventActive++
	stats.lastTouch = now

	s.registry.AddConnects(key, 1)
	s.registry.SetActive(key, stats.gauge())
	s.registry.AddTxBytes(key, txDelta)
	s.registry.AddRxBytes(key, rxDelta)
}

func (s *Store) ObserveClose(flow Flow, fallback metrics.SeriesKey, txDelta, rxDelta float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fallback
	if bound, ok := s.flows[flow]; ok {
		key = bound
		delete(s.flows, flow)
	}

	now := time.Now()
	stats := s.getOrCreateLocked(key)
	if stats.eventActive > 0 {
		stats.eventActive--
	}
	stats.lastTouch = now

	s.registry.SetActive(key, stats.gauge())
	s.registry.AddTxBytes(key, txDelta)
	s.registry.AddRxBytes(key, rxDelta)
}

// ReconcileSeedActives applies a /proc snapshot: set seedActive for unbound
// sockets, zero seedActive when those sockets disappear, and GC eBPF binds
// whose Src appeared in this snapshot but whose flow did not. Host netns is
// not walked, so those binds wait for close or connect-reuse.
func (s *Store) ReconcileSeedActives(sockets []SeedSocket) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	live := make(map[Flow]struct{}, len(sockets))
	seenSrc := make(map[netip.Addr]struct{})
	counts := make(map[metrics.SeriesKey]float64)

	for _, sock := range sockets {
		live[sock.Flow] = struct{}{}
		seenSrc[sock.Flow.Src] = struct{}{}
		if _, bound := s.flows[sock.Flow]; bound {
			continue
		}
		counts[sock.Fallback]++
	}

	for key, active := range counts {
		stats := s.getOrCreateLocked(key)
		stats.seedActive = active
		stats.lastTouch = now
		s.registry.SetActive(key, stats.gauge())
	}
	for key, stats := range s.series {
		if _, ok := counts[key]; ok {
			continue
		}
		if stats.seedActive == 0 {
			continue
		}
		stats.seedActive = 0
		stats.lastTouch = now
		s.registry.SetActive(key, stats.gauge())
	}

	for flow, key := range s.flows {
		if _, ok := live[flow]; ok {
			continue
		}
		if _, ok := seenSrc[flow.Src]; !ok {
			continue
		}
		delete(s.flows, flow)
		s.dropEventActiveLocked(key, now)
	}
}

func (s *Store) dropEventActiveLocked(key metrics.SeriesKey, now time.Time) {
	stats, ok := s.series[key]
	if !ok {
		return
	}
	if stats.eventActive > 0 {
		stats.eventActive--
	}
	stats.lastTouch = now
	s.registry.SetActive(key, stats.gauge())
}

func (s *Store) getOrCreateLocked(key metrics.SeriesKey) *seriesStats {
	if stats, ok := s.series[key]; ok {
		return stats
	}
	if len(s.series) >= s.maxSeries {
		s.evictOldestLocked()
	}
	stats := &seriesStats{lastTouch: time.Now()}
	s.series[key] = stats
	return stats
}

func (s *Store) evictOldestLocked() {
	var (
		oldestIdleKey   metrics.SeriesKey
		oldestIdleTouch time.Time
		foundIdle       bool
		oldestAnyKey    metrics.SeriesKey
		oldestAnyTouch  time.Time
		foundAny        bool
	)
	for key, stats := range s.series {
		if !foundAny || stats.lastTouch.Before(oldestAnyTouch) {
			oldestAnyKey = key
			oldestAnyTouch = stats.lastTouch
			foundAny = true
		}
		if stats.eventActive > 0 {
			continue
		}
		if !foundIdle || stats.lastTouch.Before(oldestIdleTouch) {
			oldestIdleKey = key
			oldestIdleTouch = stats.lastTouch
			foundIdle = true
		}
	}
	if !foundAny {
		return
	}
	evictKey := oldestAnyKey
	if foundIdle {
		evictKey = oldestIdleKey
	}
	for flow, key := range s.flows {
		if key == evictKey {
			delete(s.flows, flow)
		}
	}
	delete(s.series, evictKey)
	s.registry.DeleteActive(evictKey)
}
