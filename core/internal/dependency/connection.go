package dependency

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/common/model"
	"github.com/thread_koder/mochi/core/internal/prometheus"
	"golang.org/x/sync/errgroup"
)

// seriesIdentity is the shared L4 label set on mochi_net_* and mochi_http_*.
type seriesIdentity struct {
	SrcPodUID     string
	SrcNamespace  string
	SrcPod        string
	DstPodUID     string
	DstNamespace  string
	DstPod        string
	DstIP         string
	DstPort       int
	ActualDstIP   string
	ActualDstPort int
	Protocol      string
	DstHostname   string
}

// ConnectionSeries is one client-outbound connection aggregate matching the mochi_net_* label set.
type ConnectionSeries struct {
	seriesIdentity
	Connects          float64
	TxBytes           float64
	RxBytes           float64
	ActiveConnections float64
}

const (
	ProtocolTCP = "tcp"
	ProtocolUDP = "udp"
)

func isKnownProtocol(protocol string) bool {
	return protocol == ProtocolTCP || protocol == ProtocolUDP
}

func FetchConnectionSeries(ctx context.Context, opts prometheus.QueryOptions) ([]ConnectionSeries, error) {
	var (
		connects model.Vector
		txBytes  model.Vector
		rxBytes  model.Vector
		active   model.Vector
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		vector, _, err := prometheus.QueryMochiNetConnects(gctx, opts)
		if err != nil {
			return fmt.Errorf("failed to query mochi_net connects: %w", err)
		}
		connects = vector
		return nil
	})

	g.Go(func() error {
		vector, _, err := prometheus.QueryMochiNetTxBytes(gctx, opts)
		if err != nil {
			return fmt.Errorf("failed to query mochi_net tx bytes: %w", err)
		}
		txBytes = vector
		return nil
	})

	g.Go(func() error {
		vector, _, err := prometheus.QueryMochiNetRxBytes(gctx, opts)
		if err != nil {
			return fmt.Errorf("failed to query mochi_net rx bytes: %w", err)
		}
		rxBytes = vector
		return nil
	})

	g.Go(func() error {
		vector, _, err := prometheus.QueryMochiNetActiveConnections(gctx, opts)
		if err != nil {
			return fmt.Errorf("failed to query mochi_net active connections: %w", err)
		}
		active = vector
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return joinConnectionSeries(connects, txBytes, rxBytes, active), nil
}

func FetchActiveConnectionSeries(ctx context.Context, opts prometheus.QueryOptions) ([]ConnectionSeries, error) {
	active, _, err := prometheus.QueryMochiNetActiveConnections(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query mochi_net active connections: %w", err)
	}
	return joinConnectionSeries(nil, nil, nil, active), nil
}

func joinConnectionSeries(connects, txBytes, rxBytes, active model.Vector) []ConnectionSeries {
	txByKey := connectionValuesByKey(txBytes)
	rxByKey := connectionValuesByKey(rxBytes)
	activeByKey := connectionValuesByKey(active)

	seen := make(map[string]struct{}, len(connects))
	series := make([]ConnectionSeries, 0, len(connects)+len(active))

	for _, sample := range connects {
		conn, key, ok := connectionFromMetric(sample.Metric, float64(sample.Value), txByKey, rxByKey, activeByKey)
		if !ok {
			continue
		}
		seen[key] = struct{}{}
		series = append(series, conn)
	}

	// Long-lived sockets often have active > 0 with no connects increase in the window.
	for _, sample := range active {
		if float64(sample.Value) <= 0 {
			continue
		}
		conn, key, ok := connectionFromMetric(sample.Metric, 0, txByKey, rxByKey, activeByKey)
		if !ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		series = append(series, conn)
	}
	return series
}

func connectionValuesByKey(vector model.Vector) map[string]float64 {
	byKey := make(map[string]float64, len(vector))
	for _, sample := range vector {
		id, ok := parseSeriesIdentity(sample.Metric)
		if !ok {
			continue
		}
		byKey[id.l4Key()] = float64(sample.Value)
	}
	return byKey
}

func identityKey(parts ...string) string {
	return strings.Join(parts, "\x00")
}

func (id seriesIdentity) l4Key() string {
	return identityKey(
		id.SrcPodUID,
		id.SrcNamespace,
		id.SrcPod,
		id.DstPodUID,
		id.DstNamespace,
		id.DstPod,
		id.DstIP,
		strconv.Itoa(id.DstPort),
		id.ActualDstIP,
		strconv.Itoa(id.ActualDstPort),
		id.Protocol,
		id.DstHostname,
	)
}

func (id seriesIdentity) httpKey(method, route string) string {
	return identityKey(
		id.SrcPodUID,
		id.SrcNamespace,
		id.SrcPod,
		id.DstPodUID,
		id.DstNamespace,
		id.DstPod,
		id.DstIP,
		strconv.Itoa(id.DstPort),
		id.ActualDstIP,
		strconv.Itoa(id.ActualDstPort),
		id.Protocol,
		id.DstHostname,
		method,
		route,
	)
}

func parseSeriesIdentity(metric model.Metric) (seriesIdentity, bool) {
	srcPodUID := string(metric["src_pod_uid"])
	if srcPodUID == "" {
		return seriesIdentity{}, false
	}

	dstPort, err := strconv.Atoi(string(metric["dst_port"]))
	if err != nil {
		return seriesIdentity{}, false
	}

	actualDstPort, err := strconv.Atoi(string(metric["actual_dst_port"]))
	if err != nil {
		return seriesIdentity{}, false
	}

	protocol := string(metric["protocol"])
	if !isKnownProtocol(protocol) {
		return seriesIdentity{}, false
	}

	return seriesIdentity{
		SrcPodUID:     srcPodUID,
		SrcNamespace:  string(metric["src_namespace"]),
		SrcPod:        string(metric["src_pod"]),
		DstPodUID:     string(metric["dst_pod_uid"]),
		DstNamespace:  string(metric["dst_namespace"]),
		DstPod:        string(metric["dst_pod"]),
		DstIP:         string(metric["dst_ip"]),
		DstPort:       dstPort,
		ActualDstIP:   string(metric["actual_dst_ip"]),
		ActualDstPort: actualDstPort,
		Protocol:      protocol,
		DstHostname:   string(metric["dst_hostname"]),
	}, true
}

func connectionFromMetric(
	metric model.Metric,
	connects float64,
	txByKey, rxByKey, activeByKey map[string]float64,
) (ConnectionSeries, string, bool) {
	id, ok := parseSeriesIdentity(metric)
	if !ok {
		return ConnectionSeries{}, "", false
	}
	key := id.l4Key()
	return ConnectionSeries{
		seriesIdentity:    id,
		Connects:          connects,
		TxBytes:           txByKey[key],
		RxBytes:           rxByKey[key],
		ActiveConnections: activeByKey[key],
	}, key, true
}
