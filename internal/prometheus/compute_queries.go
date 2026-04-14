package prometheus

import (
	"context"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// QueryPodCPURange returns pod CPU usage over the requested range.
func QueryPodCPURange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodCPUQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryPodMemoryRange returns pod memory working set over the requested range.
func QueryPodMemoryRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodMemoryQuery(opts.Namespace, opts.Pod, opts.Container)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryPodCPUThrottling returns the pod CFS throttling ratio.
func QueryPodCPUThrottling(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildPodCPUThrottlingQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryPodCPUPressure returns the pod CPU pressure ratio.
func QueryPodCPUPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildPodCPUPressureQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryPodMemoryFailCount returns memory failcnt increases for the pod.
func QueryPodMemoryFailCount(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildPodMemoryFailCountQuery(opts.Namespace, opts.Pod, opts.Container, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryPodMemoryOOM returns OOM event increases for the pod.
func QueryPodMemoryOOM(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildPodMemoryOOMQuery(opts.Namespace, opts.Pod, opts.Container, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryPodMemoryPressure returns the pod memory pressure ratio.
func QueryPodMemoryPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildPodMemoryPressureQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryPodRestarts returns restart increases for the pod/container scope.
func QueryPodRestarts(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildPodRestartsQuery(opts.Namespace, opts.Pod, opts.Container, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryNamespaceCPURange returns namespace CPU usage over the requested range.
func QueryNamespaceCPURange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceCPUQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryNamespaceMemoryRange returns namespace memory working set over the requested range.
func QueryNamespaceMemoryRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceMemoryQuery(opts.Namespace)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryNamespaceCPUThrottling returns the namespace CFS throttling ratio.
func QueryNamespaceCPUThrottling(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceCPUThrottlingQuery(opts.Namespace, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryNamespaceCPUPressure returns the namespace CPU pressure ratio.
func QueryNamespaceCPUPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceCPUPressureQuery(opts.Namespace, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryNamespaceMemoryFailCount returns memory failcnt increases for the namespace.
func QueryNamespaceMemoryFailCount(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceMemoryFailCountQuery(opts.Namespace, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryNamespaceMemoryOOM returns OOM event increases for the namespace.
func QueryNamespaceMemoryOOM(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceMemoryOOMQuery(opts.Namespace, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryNamespaceMemoryPressure returns the namespace memory pressure ratio.
func QueryNamespaceMemoryPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceMemoryPressureQuery(opts.Namespace, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryNamespaceRestarts returns container restart increases for the namespace.
func QueryNamespaceRestarts(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceRestartsQuery(opts.Namespace, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}
