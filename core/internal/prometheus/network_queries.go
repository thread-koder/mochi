package prometheus

import (
	"context"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// QueryPodNetworkReceiveBytesRange returns pod network receive rate over the requested range.
func QueryPodNetworkReceiveBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodNetworkReceiveBytesQuery(opts.Namespace, opts.Pod, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryPodNetworkTransmitBytesRange returns pod network transmit rate over the requested range.
func QueryPodNetworkTransmitBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodNetworkTransmitBytesQuery(opts.Namespace, opts.Pod, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryPodNetworkReceiveErrorsRange returns pod receive error rate over the requested range.
func QueryPodNetworkReceiveErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodNetworkReceiveErrorsQuery(opts.Namespace, opts.Pod, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryPodNetworkTransmitErrorsRange returns pod transmit error rate over the requested range.
func QueryPodNetworkTransmitErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodNetworkTransmitErrorsQuery(opts.Namespace, opts.Pod, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryPodNetworkReceiveDroppedRange returns dropped receive packet rate over the requested range.
func QueryPodNetworkReceiveDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodNetworkReceiveDroppedQuery(opts.Namespace, opts.Pod, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// QueryPodNetworkTransmitDroppedRange returns dropped transmit packet rate over the requested range.
func QueryPodNetworkTransmitDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodNetworkTransmitDroppedQuery(opts.Namespace, opts.Pod, opts.RangeDuration)
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
