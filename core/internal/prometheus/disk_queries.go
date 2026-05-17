package prometheus

import (
	"context"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// QueryPodDiskReadBytesRange returns pod disk read byte rate over the requested range.
func QueryPodDiskReadBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodDiskReadBytesQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryPodDiskWriteBytesRange returns pod disk write byte rate over the requested range.
func QueryPodDiskWriteBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodDiskWriteBytesQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryPodDiskReadOpsRange returns pod disk read operations rate over the requested range.
func QueryPodDiskReadOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodDiskReadOpsQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryPodDiskWriteOpsRange returns pod disk write operations rate over the requested range.
func QueryPodDiskWriteOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodDiskWriteOpsQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryNamespaceDiskReadBytesRange returns namespace disk read byte rate over the requested range.
func QueryNamespaceDiskReadBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceDiskReadBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryNamespaceDiskWriteBytesRange returns namespace disk write byte rate over the requested range.
func QueryNamespaceDiskWriteBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceDiskWriteBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryNamespaceDiskReadOpsRange returns namespace disk read operations rate over the requested range.
func QueryNamespaceDiskReadOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceDiskReadOpsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryNamespaceDiskWriteOpsRange returns namespace disk write operations rate over the requested range.
func QueryNamespaceDiskWriteOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceDiskWriteOpsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}
