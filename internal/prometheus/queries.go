package prometheus

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Represents options for executing Prometheus queries
type QueryOptions struct {
	// Filters queries to a specific namespace
	Namespace string
	// Filters queries to a specific pod
	Pod string
	// Filters queries to a specific container
	Container string
	// Filters queries to a specific node
	Node string
	// Used for rate() sliding window
	RangeDuration string
}

// Executes a PromQL range query
func QueryRange(ctx context.Context, query string, r v1.Range, opts QueryOptions) (model.Value, v1.Warnings, error) {
	// Execute range query
	if API == nil {
		return nil, nil, fmt.Errorf("Prometheus API not initialized")
	}
	result, warnings, err := API.QueryRange(ctx, query, r)
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to execute PromQL range query: %w", err)
	}

	return result, warnings, nil
}

// Executes a PromQL instant query
func Query(ctx context.Context, query string, ts time.Time) (model.Value, v1.Warnings, error) {
	if API == nil {
		return nil, nil, fmt.Errorf("Prometheus API not initialized")
	}
	result, warnings, err := API.Query(ctx, query, ts)
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to execute PromQL instant query: %w", err)
	}

	return result, warnings, nil
}

// Queries pod CPU usage metrics over a time range
func QueryPodCPURange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildPodCPUQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod memory usage metrics over a time range
func QueryPodMemoryRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildPodMemoryQuery(opts.Namespace, opts.Pod, opts.Container)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod CPU throttling metrics
func QueryPodCPUThrottling(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query := BuildPodCPUThrottlingQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries pod CPU pressure metrics
func QueryPodCPUPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query := BuildPodCPUPressureQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries pod memory fail count metrics
func QueryPodMemoryFailCount(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query := BuildPodMemoryFailCountQuery(opts.Namespace, opts.Pod, opts.Container, timeRange.String())
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries pod memory OOM metrics
func QueryPodMemoryOOM(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query := BuildPodMemoryOOMQuery(opts.Namespace, opts.Pod, opts.Container, timeRange.String())
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries pod memory pressure metrics
func QueryPodMemoryPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query := BuildPodMemoryPressureQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries container restarts
func QueryContainerRestarts(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query := BuildContainerRestartsQuery(opts.Namespace, opts.Pod, opts.Container, timeRange.String())
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries namespace CPU usage metrics over a time range
func QueryNamespaceCPURange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildNamespaceCPUQuery(opts.Namespace, opts.RangeDuration)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace memory usage metrics over a time range
func QueryNamespaceMemoryRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildNamespaceMemoryQuery(opts.Namespace)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Helper to execute and cast a matrix query
func executeMatrixQuery(ctx context.Context, query string, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	result, warnings, err := QueryRange(ctx, query, r, opts)
	if err != nil {
		return nil, warnings, err
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a matrix, got %T", result)
	}

	return matrix, warnings, nil
}

// Helper to execute and cast a scalar query
func executeScalarQuery(ctx context.Context, query string, ts time.Time) (float64, v1.Warnings, error) {
	result, warnings, err := Query(ctx, query, ts)
	if err != nil {
		return 0, warnings, err
	}

	switch v := result.(type) {
	case *model.Scalar:
		return float64(v.Value), warnings, nil
	case model.Vector:
		if len(v) == 0 {
			return 0, warnings, nil
		}
		return float64(v[0].Value), warnings, nil
	default:
		return 0, warnings, fmt.Errorf("query result is not a scalar or vector, got %T", result)
	}
}
