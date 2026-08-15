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

// ConnectionSeries is one client-outbound connection aggregate matching the mochi_net_* label set.
type ConnectionSeries struct {
	SrcPodUID         string
	SrcNamespace      string
	SrcPod            string
	DstIP             string
	DstPort           int
	ActualDstIP       string
	ActualDstPort     int
	Protocol          string
	Connects          float64
	TxBytes           float64
	RxBytes           float64
	ActiveConnections float64
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

func joinConnectionSeries(connects, txBytes, rxBytes, active model.Vector) []ConnectionSeries {
	txByKey := vectorValuesByKey(txBytes)
	rxByKey := vectorValuesByKey(rxBytes)
	activeByKey := vectorValuesByKey(active)

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

func vectorValuesByKey(vector model.Vector) map[string]float64 {
	byKey := make(map[string]float64, len(vector))
	for _, sample := range vector {
		metric := sample.Metric
		byKey[identityKey(
			string(metric["src_pod_uid"]),
			string(metric["src_namespace"]),
			string(metric["src_pod"]),
			string(metric["dst_ip"]),
			string(metric["dst_port"]),
			string(metric["actual_dst_ip"]),
			string(metric["actual_dst_port"]),
			string(metric["protocol"]),
		)] = float64(sample.Value)
	}
	return byKey
}

func identityKey(parts ...string) string {
	return strings.Join(parts, "\x00")
}

func connectionFromMetric(
	metric model.Metric,
	connects float64,
	txByKey, rxByKey, activeByKey map[string]float64,
) (ConnectionSeries, string, bool) {
	srcPodUID := string(metric["src_pod_uid"])
	if srcPodUID == "" {
		return ConnectionSeries{}, "", false
	}

	dstPortLabel := string(metric["dst_port"])
	dstPort, err := strconv.Atoi(dstPortLabel)
	if err != nil {
		return ConnectionSeries{}, "", false
	}

	actualDstPortLabel := string(metric["actual_dst_port"])
	actualDstPort, err := strconv.Atoi(actualDstPortLabel)
	if err != nil {
		return ConnectionSeries{}, "", false
	}

	srcNamespace := string(metric["src_namespace"])
	srcPod := string(metric["src_pod"])
	dstIP := string(metric["dst_ip"])
	actualDstIP := string(metric["actual_dst_ip"])
	protocol := string(metric["protocol"])

	key := identityKey(
		srcPodUID,
		srcNamespace,
		srcPod,
		dstIP,
		dstPortLabel,
		actualDstIP,
		actualDstPortLabel,
		protocol,
	)

	return ConnectionSeries{
		SrcPodUID:         srcPodUID,
		SrcNamespace:      srcNamespace,
		SrcPod:            srcPod,
		DstIP:             dstIP,
		DstPort:           dstPort,
		ActualDstIP:       actualDstIP,
		ActualDstPort:     actualDstPort,
		Protocol:          protocol,
		Connects:          connects,
		TxBytes:           txByKey[key],
		RxBytes:           rxByKey[key],
		ActiveConnections: activeByKey[key],
	}, key, true
}
