package prometheus

import (
	"context"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

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
