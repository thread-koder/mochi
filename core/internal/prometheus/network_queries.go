package prometheus

import (
	"context"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

func QueryWorkloadNetworkReceiveBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkReceiveBytesQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryWorkloadNetworkTransmitBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkTransmitBytesQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryWorkloadNetworkReceiveErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkReceiveErrorsQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryWorkloadNetworkTransmitErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkTransmitErrorsQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryWorkloadNetworkReceiveDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkReceiveDroppedQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryWorkloadNetworkTransmitDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkTransmitDroppedQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceNetworkReceiveBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkReceiveBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceNetworkTransmitBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkTransmitBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceNetworkReceiveErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkReceiveErrorsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceNetworkTransmitErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkTransmitErrorsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceNetworkReceiveDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkReceiveDroppedQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceNetworkTransmitDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkTransmitDroppedQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}
