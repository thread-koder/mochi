package prometheus

import (
	"context"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

func QueryWorkloadNetworkReceiveBytes(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkReceiveBytesQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryWorkloadNetworkTransmitBytes(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkTransmitBytesQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryWorkloadNetworkReceiveErrors(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkReceiveErrorsQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryWorkloadNetworkTransmitErrors(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkTransmitErrorsQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryWorkloadNetworkReceiveDropped(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkReceiveDroppedQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryWorkloadNetworkTransmitDropped(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkTransmitDroppedQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceNetworkReceiveBytes(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkReceiveBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceNetworkTransmitBytes(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkTransmitBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceNetworkReceiveErrors(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkReceiveErrorsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceNetworkTransmitErrors(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkTransmitErrorsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceNetworkReceiveDropped(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkReceiveDroppedQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

func QueryNamespaceNetworkTransmitDropped(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkTransmitDroppedQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}
