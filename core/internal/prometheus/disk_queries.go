package prometheus

import (
	"context"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

func QueryWorkloadDiskReadBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadDiskReadBytesQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryWorkloadDiskWriteBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadDiskWriteBytesQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryWorkloadDiskReadOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadDiskReadOpsQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryWorkloadDiskWriteOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadDiskWriteOpsQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceDiskReadBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceDiskReadBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceDiskWriteBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceDiskWriteBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceDiskReadOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceDiskReadOpsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceDiskWriteOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceDiskWriteOpsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}
