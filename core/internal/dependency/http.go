package dependency

import (
	"context"
	"fmt"
	"strconv"

	"github.com/prometheus/common/model"
	"github.com/thread_koder/mochi/core/internal/prometheus"
	"golang.org/x/sync/errgroup"
)

// HTTPSeries is one client-outbound HTTP operation matching the mochi_http_* label set.
type HTTPSeries struct {
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
	Method        string
	Route         string
	Requests      float64
	Errors        float64
	P50           float64
	P95           float64
}

func FetchHTTPSeries(ctx context.Context, opts prometheus.QueryOptions) ([]HTTPSeries, error) {
	var (
		requests    model.Vector
		errorCounts model.Vector
		p50         model.Vector
		p95         model.Vector
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		vector, _, err := prometheus.QueryMochiHTTPRequests(gctx, opts)
		if err != nil {
			return fmt.Errorf("failed to query mochi_http requests: %w", err)
		}
		requests = vector
		return nil
	})

	g.Go(func() error {
		vector, _, err := prometheus.QueryMochiHTTPErrors(gctx, opts)
		if err != nil {
			return fmt.Errorf("failed to query mochi_http errors: %w", err)
		}
		errorCounts = vector
		return nil
	})

	g.Go(func() error {
		vector, _, err := prometheus.QueryMochiHTTPDurationP50(gctx, opts)
		if err != nil {
			return fmt.Errorf("failed to query mochi_http duration p50: %w", err)
		}
		p50 = vector
		return nil
	})

	g.Go(func() error {
		vector, _, err := prometheus.QueryMochiHTTPDurationP95(gctx, opts)
		if err != nil {
			return fmt.Errorf("failed to query mochi_http duration p95: %w", err)
		}
		p95 = vector
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return joinHTTPSeries(requests, errorCounts, p50, p95), nil
}

func joinHTTPSeries(requests, errorCounts, p50, p95 model.Vector) []HTTPSeries {
	errorsByKey := httpValuesByKey(errorCounts)
	p50ByKey := httpValuesByKey(p50)
	p95ByKey := httpValuesByKey(p95)

	series := make([]HTTPSeries, 0, len(requests))
	for _, sample := range requests {
		hop, ok := httpFromMetric(sample.Metric, float64(sample.Value), errorsByKey, p50ByKey, p95ByKey)
		if !ok {
			continue
		}
		series = append(series, hop)
	}
	return series
}

func httpValuesByKey(vector model.Vector) map[string]float64 {
	byKey := make(map[string]float64, len(vector))
	for _, sample := range vector {
		metric := sample.Metric
		byKey[identityKey(
			string(metric["src_pod_uid"]),
			string(metric["src_namespace"]),
			string(metric["src_pod"]),
			string(metric["dst_pod_uid"]),
			string(metric["dst_namespace"]),
			string(metric["dst_pod"]),
			string(metric["dst_ip"]),
			string(metric["dst_port"]),
			string(metric["actual_dst_ip"]),
			string(metric["actual_dst_port"]),
			string(metric["protocol"]),
			string(metric["dst_hostname"]),
			string(metric["method"]),
			string(metric["route"]),
		)] = float64(sample.Value)
	}
	return byKey
}

func httpFromMetric(
	metric model.Metric,
	requests float64,
	errorsByKey, p50ByKey, p95ByKey map[string]float64,
) (HTTPSeries, bool) {
	srcPodUID := string(metric["src_pod_uid"])
	if srcPodUID == "" {
		return HTTPSeries{}, false
	}

	method := string(metric["method"])
	if method == "" {
		return HTTPSeries{}, false
	}

	dstPortLabel := string(metric["dst_port"])
	dstPort, err := strconv.Atoi(dstPortLabel)
	if err != nil {
		return HTTPSeries{}, false
	}

	actualDstPortLabel := string(metric["actual_dst_port"])
	actualDstPort, err := strconv.Atoi(actualDstPortLabel)
	if err != nil {
		return HTTPSeries{}, false
	}

	srcNamespace := string(metric["src_namespace"])
	srcPod := string(metric["src_pod"])
	dstPodUID := string(metric["dst_pod_uid"])
	dstNamespace := string(metric["dst_namespace"])
	dstPod := string(metric["dst_pod"])
	dstIP := string(metric["dst_ip"])
	actualDstIP := string(metric["actual_dst_ip"])
	protocol := string(metric["protocol"])
	dstHostname := string(metric["dst_hostname"])
	route := string(metric["route"])
	if !isKnownProtocol(protocol) {
		return HTTPSeries{}, false
	}

	key := identityKey(
		srcPodUID,
		srcNamespace,
		srcPod,
		dstPodUID,
		dstNamespace,
		dstPod,
		dstIP,
		dstPortLabel,
		actualDstIP,
		actualDstPortLabel,
		protocol,
		dstHostname,
		method,
		route,
	)

	return HTTPSeries{
		SrcPodUID:     srcPodUID,
		SrcNamespace:  srcNamespace,
		SrcPod:        srcPod,
		DstPodUID:     dstPodUID,
		DstNamespace:  dstNamespace,
		DstPod:        dstPod,
		DstIP:         dstIP,
		DstPort:       dstPort,
		ActualDstIP:   actualDstIP,
		ActualDstPort: actualDstPort,
		Protocol:      protocol,
		DstHostname:   dstHostname,
		Method:        method,
		Route:         route,
		Requests:      requests,
		Errors:        errorsByKey[key],
		P50:           p50ByKey[key],
		P95:           p95ByKey[key],
	}, true
}
