package prometheus

import (
	"context"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// QueryWorkloadNetworkReceiveBytesRange returns workload network receive rate over the requested range.
func QueryWorkloadNetworkReceiveBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkReceiveBytesQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryWorkloadNetworkTransmitBytesRange returns workload network transmit rate over the requested range.
func QueryWorkloadNetworkTransmitBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkTransmitBytesQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryWorkloadNetworkReceiveErrorsRange returns workload receive error rate over the requested range.
func QueryWorkloadNetworkReceiveErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkReceiveErrorsQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryWorkloadNetworkTransmitErrorsRange returns workload transmit error rate over the requested range.
func QueryWorkloadNetworkTransmitErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkTransmitErrorsQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryWorkloadNetworkReceiveDroppedRange returns dropped receive packet rate over the requested range.
func QueryWorkloadNetworkReceiveDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkReceiveDroppedQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryWorkloadNetworkTransmitDroppedRange returns dropped transmit packet rate over the requested range.
func QueryWorkloadNetworkTransmitDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildWorkloadNetworkTransmitDroppedQuery(opts.Namespace, opts.Pods, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryNamespaceNetworkReceiveBytesRange returns namespace network receive rate over the requested range.
func QueryNamespaceNetworkReceiveBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkReceiveBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryNamespaceNetworkTransmitBytesRange returns namespace network transmit rate over the requested range.
func QueryNamespaceNetworkTransmitBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkTransmitBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryNamespaceNetworkReceiveErrorsRange returns namespace receive error rate over the requested range.
func QueryNamespaceNetworkReceiveErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkReceiveErrorsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryNamespaceNetworkTransmitErrorsRange returns namespace transmit error rate over the requested range.
func QueryNamespaceNetworkTransmitErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkTransmitErrorsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryNamespaceNetworkReceiveDroppedRange returns dropped receive packet rate over the requested range.
func QueryNamespaceNetworkReceiveDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkReceiveDroppedQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryNamespaceNetworkTransmitDroppedRange returns dropped transmit packet rate over the requested range.
func QueryNamespaceNetworkTransmitDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkTransmitDroppedQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}
