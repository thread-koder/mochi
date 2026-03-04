package prometheus

import (
	"context"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Queries pod network receive bytes metrics over a time range (bytes/sec)
func QueryPodNetworkReceiveBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodNetworkReceiveBytesQuery(opts.Namespace, opts.Pod, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod network transmit bytes metrics over a time range (bytes/sec)
func QueryPodNetworkTransmitBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodNetworkTransmitBytesQuery(opts.Namespace, opts.Pod, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod network receive errors metrics over a time range (errors/sec)
func QueryPodNetworkReceiveErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodNetworkReceiveErrorsQuery(opts.Namespace, opts.Pod, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod network transmit errors metrics over a time range (errors/sec)
func QueryPodNetworkTransmitErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodNetworkTransmitErrorsQuery(opts.Namespace, opts.Pod, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod network receive dropped packets metrics over a time range (packets/sec)
func QueryPodNetworkReceiveDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodNetworkReceiveDroppedQuery(opts.Namespace, opts.Pod, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod network transmit dropped packets metrics over a time range (packets/sec)
func QueryPodNetworkTransmitDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildPodNetworkTransmitDroppedQuery(opts.Namespace, opts.Pod, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace network receive bytes metrics over a time range (bytes/sec)
func QueryNamespaceNetworkReceiveBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkReceiveBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace network transmit bytes metrics over a time range (bytes/sec)
func QueryNamespaceNetworkTransmitBytesRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkTransmitBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace network receive errors metrics over a time range (errors/sec)
func QueryNamespaceNetworkReceiveErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkReceiveErrorsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace network transmit errors metrics over a time range (errors/sec)
func QueryNamespaceNetworkTransmitErrorsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkTransmitErrorsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace network receive dropped packets metrics over a time range (packets/sec)
func QueryNamespaceNetworkReceiveDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkReceiveDroppedQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace network transmit dropped packets metrics over a time range (packets/sec)
func QueryNamespaceNetworkTransmitDroppedRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query, err := BuildNamespaceNetworkTransmitDroppedQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeMatrixQuery(ctx, query, r, opts)
}
