package prometheus

import (
	"context"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

func QueryMochiNetConnects(ctx context.Context, opts QueryOptions) (model.Vector, v1.Warnings, error) {
	query, err := BuildMochiNetConnectsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeVectorQuery(ctx, query, time.Now())
}

func QueryMochiNetTxBytes(ctx context.Context, opts QueryOptions) (model.Vector, v1.Warnings, error) {
	query, err := BuildMochiNetTxBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeVectorQuery(ctx, query, time.Now())
}

func QueryMochiNetRxBytes(ctx context.Context, opts QueryOptions) (model.Vector, v1.Warnings, error) {
	query, err := BuildMochiNetRxBytesQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeVectorQuery(ctx, query, time.Now())
}

func QueryMochiNetActiveConnections(ctx context.Context, opts QueryOptions) (model.Vector, v1.Warnings, error) {
	query, err := BuildMochiNetActiveConnectionsQuery(opts.Namespace)
	if err != nil {
		return nil, nil, err
	}
	return executeVectorQuery(ctx, query, time.Now())
}
