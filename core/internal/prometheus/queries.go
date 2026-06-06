package prometheus

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// QueryOptions controls label filters and range windows for query builders.
type QueryOptions struct {
	// Namespace scopes queries to a Kubernetes namespace when set.
	Namespace string
	// Pods scopes queries to one or more Kubernetes pods when set.
	Pods []string
	// Container scopes queries to a single container when set.
	Container string
	// Node is reserved for node-scoped metrics.
	Node string
	// RangeDuration is the rate/increase lookback window (for example "5m").
	RangeDuration string
}

// QueryRange executes a PromQL range query given the query string, range, and options.
func QueryRange(ctx context.Context, query string, r v1.Range, opts QueryOptions) (model.Value, v1.Warnings, error) {
	if API == nil {
		return nil, nil, fmt.Errorf("prometheus api not initialized")
	}
	result, warnings, err := API.QueryRange(ctx, query, r)
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to execute PromQL range query: %w", err)
	}

	return result, warnings, nil
}

// Query executes a PromQL instant query given the query string and timestamp.
func Query(ctx context.Context, query string, ts time.Time) (model.Value, v1.Warnings, error) {
	if API == nil {
		return nil, nil, fmt.Errorf("prometheus api not initialized")
	}
	result, warnings, err := API.Query(ctx, query, ts)
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to execute PromQL instant query: %w", err)
	}

	return result, warnings, nil
}

// executeMatrixQuery executes a range query and enforces a matrix result type.
func executeMatrixQuery(ctx context.Context, query string, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	result, warnings, err := QueryRange(ctx, query, r, opts)
	if err != nil {
		return nil, warnings, err
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a matrix, got %T", result)
	}

	return matrix, warnings, nil
}

// executeScalarQuery executes an instant query and extracts one scalar value.
//
// Some PromQL expressions that are scalar in intent return a one-sample vector,
// so this function accepts both scalar and 0/1-length vector responses.
func executeScalarQuery(ctx context.Context, query string, ts time.Time) (float64, v1.Warnings, error) {
	result, warnings, err := Query(ctx, query, ts)
	if err != nil {
		return 0, warnings, err
	}

	switch v := result.(type) {
	case *model.Scalar:
		return float64(v.Value), warnings, nil
	case model.Vector:
		if len(v) == 0 {
			return 0, warnings, nil
		}
		if len(v) > 1 {
			return 0, warnings, fmt.Errorf("scalar query returned %d series, expected 0 or 1", len(v))
		}
		return float64(v[0].Value), warnings, nil
	default:
		return 0, warnings, fmt.Errorf("query result is not a scalar or vector, got %T", result)
	}
}
