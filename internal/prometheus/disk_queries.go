package prometheus

import (
	"context"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Queries pod disk read bytes metrics over a time range (bytes/sec)
func QueryPodDiskReadBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodDiskReadBytesQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod disk write bytes metrics over a time range (bytes/sec)
func QueryPodDiskWriteBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodDiskWriteBytesQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod disk read operations metrics over a time range (IOPS)
func QueryPodDiskReadOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodDiskReadOpsQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod disk write operations metrics over a time range (IOPS)
func QueryPodDiskWriteOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodDiskWriteOpsQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace disk read bytes metrics over a time range (bytes/sec)
func QueryNamespaceDiskReadBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceDiskReadBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace disk write bytes metrics over a time range (bytes/sec)
func QueryNamespaceDiskWriteBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceDiskWriteBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace disk read operations metrics over a time range (IOPS)
func QueryNamespaceDiskReadOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceDiskReadOpsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace disk write operations metrics over a time range (IOPS)
func QueryNamespaceDiskWriteOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceDiskWriteOpsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}
