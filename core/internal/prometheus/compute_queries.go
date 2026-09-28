package prometheus

import (
	"context"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

func QueryWorkloadCPUUsage(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadCPUUsageQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r)
}

func QueryWorkloadMemoryUsage(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadMemoryUsageQuery(opts.Namespace, opts.Pods, opts.Container)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r)
}

func QueryWorkloadCPUThrottling(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildWorkloadCPUThrottlingQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, opts.evalAt())
}

func QueryWorkloadCPUPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildWorkloadCPUPressureQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, opts.evalAt())
}

func QueryWorkloadMemoryFailCount(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildWorkloadMemoryFailCountQuery(opts.Namespace, opts.Pods, opts.Container, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, opts.evalAt())
}

func QueryWorkloadMemoryOOM(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildWorkloadMemoryOOMQuery(opts.Namespace, opts.Pods, opts.Container, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, opts.evalAt())
}

func QueryWorkloadMemoryPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildWorkloadMemoryPressureQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, opts.evalAt())
}

func QueryWorkloadRestarts(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildWorkloadRestartsQuery(opts.Namespace, opts.Pods, opts.Container, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, opts.evalAt())
}

func QueryNamespaceCPUUsage(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceCPUUsageQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r)
}

func QueryNamespaceMemoryUsage(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceMemoryUsageQuery(opts.Namespace)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r)
}

func QueryNamespaceCPUThrottling(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceCPUThrottlingQuery(opts.Namespace, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, opts.evalAt())
}

func QueryNamespaceCPUPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceCPUPressureQuery(opts.Namespace, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, opts.evalAt())
}

func QueryNamespaceMemoryFailCount(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceMemoryFailCountQuery(opts.Namespace, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, opts.evalAt())
}

func QueryNamespaceMemoryOOM(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceMemoryOOMQuery(opts.Namespace, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, opts.evalAt())
}

func QueryNamespaceMemoryPressure(ctx context.Context, timeRange time.Duration, step time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceMemoryPressureQuery(opts.Namespace, opts.RangeDuration, timeRange.String(), step.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, opts.evalAt())
}

func QueryNamespaceRestarts(ctx context.Context, timeRange time.Duration, opts QueryOptions) (float64, v1.Warnings, error) {
	query, err := BuildNamespaceRestartsQuery(opts.Namespace, timeRange.String())
	if err != nil {
		return 0, nil, err
	}
	return executeScalarQuery(ctx, query, opts.evalAt())
}
