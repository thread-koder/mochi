package dependency

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/prometheus/common/model"
	"github.com/thread_koder/mochi/core/internal/prometheus"
	"golang.org/x/sync/errgroup"
)

// HTTPSeries is one client-outbound HTTP operation matching the mochi_http_* label set.
type HTTPSeries struct {
	seriesIdentity
	Method   string
	Route    string
	Requests float64
	Errors   float64
	P50      float64
	P95      float64
}

// connectionSeries adapts L4 identity for Resolve. Connects carries request volume for the volume gate.
func (series HTTPSeries) connectionSeries() ConnectionSeries {
	return ConnectionSeries{
		seriesIdentity: series.seriesIdentity,
		Connects:       series.Requests,
	}
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
		id, ok := parseSeriesIdentity(sample.Metric)
		if !ok {
			continue
		}
		method := string(sample.Metric["method"])
		route := string(sample.Metric["route"])
		byKey[id.httpKey(method, route)] = float64(sample.Value)
	}
	return byKey
}

func httpFromMetric(
	metric model.Metric,
	requests float64,
	errorsByKey, p50ByKey, p95ByKey map[string]float64,
) (HTTPSeries, bool) {
	id, ok := parseSeriesIdentity(metric)
	if !ok {
		return HTTPSeries{}, false
	}

	method := string(metric["method"])
	if method == "" {
		return HTTPSeries{}, false
	}
	route := string(metric["route"])
	key := id.httpKey(method, route)

	p50 := math.NaN()
	if value, ok := p50ByKey[key]; ok {
		p50 = value
	}
	p95 := math.NaN()
	if value, ok := p95ByKey[key]; ok {
		p95 = value
	}

	return HTTPSeries{
		seriesIdentity: id,
		Method:         method,
		Route:          route,
		Requests:       requests,
		Errors:         errorsByKey[key],
		P50:            p50,
		P95:            p95,
	}, true
}

type httpEdge struct {
	Requests   float64
	Errors     float64
	Operations map[string]*httpOp
}

type httpOp struct {
	Method           string
	Route            string
	Requests         float64
	Errors           float64
	P50              float64
	P95              float64
	dominantRequests float64
}

func operationKey(method, route string) string {
	return method + "\x00" + route
}

func mergeHTTPByEdge(ctx context.Context, series []HTTPSeries, opts ResolveOptions) (map[string]*httpEdge, error) {
	byEdge := make(map[string]*httpEdge)
	for _, hop := range series {
		from, to, kept, err := resolveEdgeEnds(ctx, hop.connectionSeries(), opts)
		if err != nil {
			return nil, fmt.Errorf("resolve http series: %w", err)
		}
		if !kept {
			continue
		}

		key := edgeKey(ResolvedEdge{
			From:     from,
			To:       to,
			Protocol: hop.Protocol,
			Port:     hop.ActualDstPort,
		})
		edge, ok := byEdge[key]
		if !ok {
			edge = &httpEdge{Operations: make(map[string]*httpOp)}
			byEdge[key] = edge
		}
		edge.Requests += hop.Requests
		edge.Errors += hop.Errors

		opKey := operationKey(hop.Method, hop.Route)
		op, ok := edge.Operations[opKey]
		if !ok {
			op = &httpOp{Method: hop.Method, Route: hop.Route}
			edge.Operations[opKey] = op
		}
		op.Requests += hop.Requests
		op.Errors += hop.Errors
		// Quantiles are not additive. Keep the series with the most requests on this op.
		if hop.Requests > op.dominantRequests {
			op.dominantRequests = hop.Requests
			op.P50 = hop.P50
			op.P95 = hop.P95
		}
	}
	return byEdge, nil
}

func finiteQuantile(v float64) *float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nil
	}
	return new(v)
}

func (edge *httpEdge) toOperations() []OperationDTO {
	ops := make([]OperationDTO, 0, len(edge.Operations))
	for _, op := range edge.Operations {
		ops = append(ops, OperationDTO{
			Method:   op.Method,
			Route:    op.Route,
			Requests: op.Requests,
			Errors:   op.Errors,
			P50:      finiteQuantile(op.P50),
			P95:      finiteQuantile(op.P95),
		})
	}
	slices.SortFunc(ops, func(a, b OperationDTO) int {
		if c := cmp.Compare(b.Requests, a.Requests); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Method, b.Method); c != 0 {
			return c
		}
		return cmp.Compare(a.Route, b.Route)
	})
	return ops
}

func (edge *httpEdge) dominantOpQuantiles() (p50, p95 *float64) {
	var best *httpOp
	for _, op := range edge.Operations {
		if best == nil || op.Requests > best.Requests {
			best = op
		}
	}
	if best == nil {
		return nil, nil
	}
	return finiteQuantile(best.P50), finiteQuantile(best.P95)
}
