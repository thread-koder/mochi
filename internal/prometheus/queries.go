package prometheus

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Represents options for executing Prometheus queries
type QueryOptions struct {
	// Filters queries to a specific namespace
	Namespace string
	// Filters queries to a specific pod
	Pod string
	// Filters queries to a specific container
	Container string
	// Filters queries to a specific node
	Node string
	// Used for rate() sliding window
	RangeDuration string
}

// Executes a PromQL range query
func QueryRange(ctx context.Context, query string, r v1.Range, opts QueryOptions) (model.Value, v1.Warnings, error) {
	// Execute range query
	if API == nil {
		return nil, nil, fmt.Errorf("Prometheus API not initialized")
	}
	result, warnings, err := API.QueryRange(ctx, query, r)
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to execute PromQL range query: %w", err)
	}

	return result, warnings, nil
}

// Executes a PromQL instant query
func Query(ctx context.Context, query string, ts time.Time) (model.Value, v1.Warnings, error) {
	if API == nil {
		return nil, nil, fmt.Errorf("Prometheus API not initialized")
	}
	result, warnings, err := API.Query(ctx, query, ts)
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to execute PromQL instant query: %w", err)
	}

	return result, warnings, nil
}

// Executes a range query and returns the result as a matrix
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

// Helper to execute and cast a scalar query
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
