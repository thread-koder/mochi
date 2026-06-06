package prometheus

import (
	"context"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// QueryWorkloadDiskReadBytesRange returns workload disk read byte rate over the requested range.
func QueryWorkloadDiskReadBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadDiskReadBytesQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryWorkloadDiskWriteBytesRange returns workload disk write byte rate over the requested range.
func QueryWorkloadDiskWriteBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadDiskWriteBytesQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryWorkloadDiskReadOpsRange returns workload disk read operations rate over the requested range.
func QueryWorkloadDiskReadOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadDiskReadOpsQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryWorkloadDiskWriteOpsRange returns workload disk write operations rate over the requested range.
func QueryWorkloadDiskWriteOpsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadDiskWriteOpsQuery(opts.Namespace, opts.Pods, opts.Container, opts.RangeDuration)
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
