package prometheus

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Executes a PromQL query and returns the result
func Query(ctx context.Context, query string, timestamp time.Time) (model.Value, v1.Warnings, error) {
	if API == nil {
		return nil, nil, fmt.Errorf("Prometheus API not initialized")
	}

	result, warnings, err := API.Query(ctx, query, timestamp)
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to execute PromQL query: %w", err)
	}

	return result, warnings, nil
}

// Executes a PromQL query at the current time
func QueryNow(ctx context.Context, query string) (model.Value, v1.Warnings, error) {
	return Query(ctx, query, time.Now())
}

// Executes a PromQL range query and returns the result
func QueryRange(ctx context.Context, query string, r v1.Range) (model.Value, v1.Warnings, error) {
	if API == nil {
		return nil, nil, fmt.Errorf("Prometheus API not initialized")
	}

	result, warnings, err := API.QueryRange(ctx, query, r)
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to execute PromQL range query: %w", err)
	}

	return result, warnings, nil
}

// Creates a range query configuration
func NewRange(start, end time.Time, step time.Duration) v1.Range {
	return v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	}
}

// Executes a PromQL query and returns the result as a vector
func QueryVector(ctx context.Context, query string) (model.Vector, v1.Warnings, error) {
	result, warnings, err := QueryNow(ctx, query)
	if err != nil {
		return nil, warnings, err
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a vector, got %T", result)
	}

	return vector, warnings, nil
}

// Executes a PromQL query and returns the result as a matrix (for range queries)
func QueryMatrix(ctx context.Context, query string, r v1.Range) (model.Matrix, v1.Warnings, error) {
	result, warnings, err := QueryRange(ctx, query, r)
	if err != nil {
		return nil, warnings, err
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a matrix, got %T", result)
	}

	return matrix, warnings, nil
}

// Executes a PromQL query and returns the result as a scalar
func QueryScalar(ctx context.Context, query string) (*model.Scalar, v1.Warnings, error) {
	result, warnings, err := QueryNow(ctx, query)
	if err != nil {
		return nil, warnings, err
	}

	scalar, ok := result.(*model.Scalar)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a scalar, got %T", result)
	}

	return scalar, warnings, nil
}

// Executes a PromQL query and returns the result as a string
func QueryString(ctx context.Context, query string) (*model.String, v1.Warnings, error) {
	result, warnings, err := QueryNow(ctx, query)
	if err != nil {
		return nil, warnings, err
	}

	str, ok := result.(*model.String)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a string, got %T", result)
	}

	return str, warnings, nil
}
