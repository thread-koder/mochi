package prometheus

import (
	"context"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// QueryWorkloadCPURange returns workload CPU usage over the requested range.
func QueryWorkloadCPURange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadCPUQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryWorkloadMemoryRange returns workload memory working set over the requested range.
func QueryWorkloadMemoryRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadMemoryQuery(opts.Namespace, opts.Pods, opts.Container)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryWorkloadCPUThrottling returns the workload CFS throttling ratio.
func QueryWorkloadCPUThrottling(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildWorkloadCPUThrottlingQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryWorkloadCPUPressure returns the workload CPU pressure ratio.
func QueryWorkloadCPUPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildWorkloadCPUPressureQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryWorkloadMemoryFailCount returns memory failcnt increases for the workload.
func QueryWorkloadMemoryFailCount(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildWorkloadMemoryFailCountQuery(opts.Namespace, opts.Pods, opts.Container, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryWorkloadMemoryOOM returns OOM event increases for the workload.
func QueryWorkloadMemoryOOM(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildWorkloadMemoryOOMQuery(opts.Namespace, opts.Pods, opts.Container, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryWorkloadMemoryPressure returns the workload memory pressure ratio.
func QueryWorkloadMemoryPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildWorkloadMemoryPressureQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, time.Now())
}

// QueryWorkloadRestarts returns restart increases for the workload/container scope.
func QueryWorkloadRestarts(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildWorkloadRestartsQuery(opts.Namespace, opts.Pods, opts.Container, timeRange.String())
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
