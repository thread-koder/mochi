package prometheus

import (
	"context"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

func QueryMochiHTTPRequests(ctx context.Context, opts QueryOptions) (model.Vector, v1.Warnings, error) {
	query, err := BuildMochiHTTPRequestsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeVectorQuery(ctx, query, opts.evalAt())
}

func QueryMochiHTTPErrors(ctx context.Context, opts QueryOptions) (model.Vector, v1.Warnings, error) {
	query, err := BuildMochiHTTPErrorsQuery(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeVectorQuery(ctx, query, opts.evalAt())
}

func QueryMochiHTTPDurationP50(ctx context.Context, opts QueryOptions) (model.Vector, v1.Warnings, error) {
	query, err := BuildMochiHTTPDurationP50Query(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeVectorQuery(ctx, query, opts.evalAt())
}

func QueryMochiHTTPDurationP95(ctx context.Context, opts QueryOptions) (model.Vector, v1.Warnings, error) {
	query, err := BuildMochiHTTPDurationP95Query(opts.Namespace, opts.RangeDuration)
	if err != nil {
		return nil, nil, err
	}
	return executeVectorQuery(ctx, query, opts.evalAt())
}
