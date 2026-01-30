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
	// Used for "total over period" queries (restarts, OOM, memory fail).
	AnalysisRange string
}

// Represents pod-level metric results
type PodMetricResult struct {
	Pod       string    `json:"pod"`
	Namespace string    `json:"namespace"`
	Container string    `json:"container"`
	Node      string    `json:"node"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// Represents namespace-level aggregated metric results
type NamespaceMetricResult struct {
	Namespace string    `json:"namespace"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
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

// Queries pod CPU usage metrics over a time range
func QueryPodCPURange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildPodCPUQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod memory usage metrics over a time range
func QueryPodMemoryRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildPodMemoryQuery(opts.Namespace, opts.Pod, opts.Container)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod CPU throttling metrics over a time range
func QueryPodCPUThrottlingRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildPodCPUThrottlingQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod CPU pressure metrics over a time range
func QueryPodCPUPressureRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildPodCPUPressureQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod memory fail count metrics over a time range
func QueryPodMemoryFailCountRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildPodMemoryFailCountQuery(opts.Namespace, opts.Pod, opts.Container, opts.AnalysisRange)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod memory OOM metrics over a time range
func QueryPodMemoryOOMRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildPodMemoryOOMQuery(opts.Namespace, opts.Pod, opts.Container, opts.AnalysisRange)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries pod memory pressure metrics over a time range
func QueryPodMemoryPressureRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildPodMemoryPressureQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries container restarts over a time range
func QueryContainerRestartsRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildContainerRestartsQuery(opts.Namespace, opts.Pod, opts.Container, opts.AnalysisRange)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace CPU usage metrics over a time range (aggregated)
func QueryNamespaceCPURange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildNamespaceCPUQuery(opts.Namespace, opts.RangeDuration)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Queries namespace memory usage metrics over a time range (aggregated)
func QueryNamespaceMemoryRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildNamespaceMemoryQuery(opts.Namespace)
	return executeMatrixQuery(ctx, query, r, opts)
}

// Helper to execute and cast a matrix query
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
