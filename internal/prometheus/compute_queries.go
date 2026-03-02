package prometheus

import (
	"context"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Queries pod CPU usage metrics over a time range
func QueryPodCPURange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodCPUQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod memory usage metrics over a time range
func QueryPodMemoryRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodMemoryQuery(opts.Namespace, opts.Pod, opts.Container)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod CPU throttling metrics
func QueryPodCPUThrottling(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildPodCPUThrottlingQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries pod CPU pressure metrics
func QueryPodCPUPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildPodCPUPressureQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries pod memory fail count metrics
func QueryPodMemoryFailCount(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildPodMemoryFailCountQuery(opts.Namespace, opts.Pod, opts.Container, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries pod memory OOM metrics
func QueryPodMemoryOOM(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildPodMemoryOOMQuery(opts.Namespace, opts.Pod, opts.Container, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries pod memory pressure metrics
func QueryPodMemoryPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildPodMemoryPressureQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries pod restarts
func QueryPodRestarts(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildPodRestartsQuery(opts.Namespace, opts.Pod, opts.Container, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries namespace CPU usage metrics over a time range
func QueryNamespaceCPURange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceCPUQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace memory usage metrics over a time range
func QueryNamespaceMemoryRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceMemoryQuery(opts.Namespace)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace CPU throttling metrics
func QueryNamespaceCPUThrottling(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceCPUThrottlingQuery(opts.Namespace, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries namespace CPU pressure metrics
func QueryNamespaceCPUPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceCPUPressureQuery(opts.Namespace, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries namespace memory fail count metrics
func QueryNamespaceMemoryFailCount(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceMemoryFailCountQuery(opts.Namespace, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries namespace memory OOM metrics
func QueryNamespaceMemoryOOM(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceMemoryOOMQuery(opts.Namespace, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries namespace memory pressure metrics
func QueryNamespaceMemoryPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceMemoryPressureQuery(opts.Namespace, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// Queries namespace container restarts
func QueryNamespaceRestarts(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceRestartsQuery(opts.Namespace, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}
