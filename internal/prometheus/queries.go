package prometheus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	rediscache "github.com/redis/go-redis/v9"
	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/logger"
	"github.com/thread_koder/mochi/internal/redis"
)

// Holds options for executing Prometheus queries
type QueryOptions struct {
	// Determines if the query result should be cached
	UseCache bool
	// The time-to-live for cached results (overrides default if set)
	CacheTTL time.Duration
	// Filters queries to a specific namespace
	Namespace string
	// Filters queries to a specific pod
	Pod string
	// Filters queries to a specific container
	Container string
	// Filters queries to a specific node
	Node string
	// The duration for range queries (e.g., "5m", "1h")
	RangeDuration string
}

// Holds pod-level metric results
type PodMetricResult struct {
	Pod       string
	Namespace string
	Container string
	Node      string
	Value     float64
	Timestamp time.Time
}

// Holds node-level metric results
type NodeMetricResult struct {
	Node      string
	Value     float64
	Timestamp time.Time
}

// Holds namespace-level aggregated metric results
type NamespaceMetricResult struct {
	Namespace string
	Value     float64
	Timestamp time.Time
}

// Builds a cache key from a query string
func buildCacheKey(query string) string {
	hash := sha256.Sum256([]byte(query))
	return fmt.Sprintf("prometheus:query:%s", hex.EncodeToString(hash[:]))
}

// Executes a PromQL query with optional caching
func QueryWithCache(ctx context.Context, query string, opts QueryOptions) (model.Value, v1.Warnings, error) {
	log := logger.WithComponent("prometheus")

	// Check cache if enabled
	if opts.UseCache {
		cacheKey := buildCacheKey(query)
		var cachedResult model.Value

		// Try to get from cache
		if err := redis.GetJSON(ctx, cacheKey, &cachedResult); err == nil {
			log.Debug().
				Str("query", query).
				Str("cache_key", cacheKey).
				Msg("Cache hit")
			return cachedResult, nil, nil
		} else if !errors.Is(err, rediscache.Nil) {
			log.Warn().
				Err(err).
				Str("cache_key", cacheKey).
				Msg("Failed to read from cache, executing query")
		}
	}

	// Execute query
	if API == nil {
		return nil, nil, fmt.Errorf("Prometheus API not initialized")
	}
	result, warnings, err := API.Query(ctx, query, time.Now())
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to execute PromQL query: %w", err)
	}

	// Cache result if enabled
	if opts.UseCache {
		cacheKey := buildCacheKey(query)
		ttl := opts.CacheTTL
		if ttl == 0 {
			// Use default TTL from config
			cfg := config.AppConfig
			if cfg != nil {
				ttl = redis.GetDefaultTTL(&cfg.Redis)
			} else {
				ttl = 5 * time.Minute // Fallback default
			}
		}

		if err := redis.SetJSON(ctx, cacheKey, result, ttl); err != nil {
			log.Warn().
				Err(err).
				Str("cache_key", cacheKey).
				Msg("Failed to cache query result")
		} else {
			log.Debug().
				Str("query", query).
				Str("cache_key", cacheKey).
				Dur("ttl", ttl).
				Msg("Cached query result")
		}
	}

	return result, warnings, nil
}

// Executes a PromQL range query with optional caching
func QueryRangeWithCache(ctx context.Context, query string, r v1.Range, opts QueryOptions) (model.Value, v1.Warnings, error) {
	log := logger.WithComponent("prometheus")

	// For range queries, include time range in cache key to ensure uniqueness
	cacheKey := buildCacheKey(fmt.Sprintf("%s:%d:%d:%d", query, r.Start.Unix(), r.End.Unix(), r.Step.Nanoseconds()))

	// Check cache if enabled
	if opts.UseCache {
		var cachedResult model.Value

		// Try to get from cache
		if err := redis.GetJSON(ctx, cacheKey, &cachedResult); err == nil {
			log.Debug().
				Str("query", query).
				Str("cache_key", cacheKey).
				Msg("Cache hit for range query")
			return cachedResult, nil, nil
		} else if !errors.Is(err, rediscache.Nil) {
			log.Warn().
				Err(err).
				Str("cache_key", cacheKey).
				Msg("Failed to read from cache, executing range query")
		}
	}

	// Execute range query
	if API == nil {
		return nil, nil, fmt.Errorf("Prometheus API not initialized")
	}
	result, warnings, err := API.QueryRange(ctx, query, r)
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to execute PromQL range query: %w", err)
	}

	// Cache result if enabled
	if opts.UseCache {
		ttl := opts.CacheTTL
		if ttl == 0 {
			// Use default TTL from config
			cfg := config.AppConfig
			if cfg != nil {
				ttl = redis.GetDefaultTTL(&cfg.Redis)
			} else {
				ttl = 5 * time.Minute // Fallback default
			}
		}

		if err := redis.SetJSON(ctx, cacheKey, result, ttl); err != nil {
			log.Warn().
				Err(err).
				Str("cache_key", cacheKey).
				Msg("Failed to cache range query result")
		} else {
			log.Debug().
				Str("query", query).
				Str("cache_key", cacheKey).
				Dur("ttl", ttl).
				Msg("Cached range query result")
		}
	}

	return result, warnings, nil
}

// Queries pod CPU usage metrics
func QueryPodCPU(ctx context.Context, opts QueryOptions) ([]PodMetricResult, v1.Warnings, error) {
	query := BuildPodCPUQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)

	result, warnings, err := QueryWithCache(ctx, query, opts)
	if err != nil {
		return nil, warnings, err
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a vector, got %T", result)
	}

	results := make([]PodMetricResult, 0, len(vector))
	for _, sample := range vector {
		podName := string(sample.Metric["pod"])
		namespace := string(sample.Metric["namespace"])
		container := string(sample.Metric["container"])
		node := string(sample.Metric["node"])

		results = append(results, PodMetricResult{
			Pod:       podName,
			Namespace: namespace,
			Container: container,
			Node:      node,
			Value:     float64(sample.Value),
			Timestamp: sample.Timestamp.Time(),
		})
	}

	return results, warnings, nil
}

// Queries pod memory usage metrics
func QueryPodMemory(ctx context.Context, opts QueryOptions) ([]PodMetricResult, v1.Warnings, error) {
	query := BuildPodMemoryQuery(opts.Namespace, opts.Pod, opts.Container)

	result, warnings, err := QueryWithCache(ctx, query, opts)
	if err != nil {
		return nil, warnings, err
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a vector, got %T", result)
	}

	results := make([]PodMetricResult, 0, len(vector))
	for _, sample := range vector {
		podName := string(sample.Metric["pod"])
		namespace := string(sample.Metric["namespace"])
		container := string(sample.Metric["container"])
		node := string(sample.Metric["node"])

		results = append(results, PodMetricResult{
			Pod:       podName,
			Namespace: namespace,
			Container: container,
			Node:      node,
			Value:     float64(sample.Value),
			Timestamp: sample.Timestamp.Time(),
		})
	}

	return results, warnings, nil
}

// Queries node CPU usage metrics
func QueryNodeCPU(ctx context.Context, opts QueryOptions) ([]NodeMetricResult, v1.Warnings, error) {
	query := BuildNodeCPUQuery(opts.Node, opts.RangeDuration)

	result, warnings, err := QueryWithCache(ctx, query, opts)
	if err != nil {
		return nil, warnings, err
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a vector, got %T", result)
	}

	results := make([]NodeMetricResult, 0, len(vector))
	for _, sample := range vector {
		node := string(sample.Metric["instance"])
		if node == "" {
			node = opts.Node
		}

		results = append(results, NodeMetricResult{
			Node:      node,
			Value:     float64(sample.Value),
			Timestamp: sample.Timestamp.Time(),
		})
	}

	return results, warnings, nil
}

// Queries node memory usage metrics
func QueryNodeMemory(ctx context.Context, opts QueryOptions) ([]NodeMetricResult, v1.Warnings, error) {
	query := BuildNodeMemoryQuery(opts.Node)

	result, warnings, err := QueryWithCache(ctx, query, opts)
	if err != nil {
		return nil, warnings, err
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a vector, got %T", result)
	}

	results := make([]NodeMetricResult, 0, len(vector))
	for _, sample := range vector {
		node := string(sample.Metric["instance"])
		if node == "" {
			node = opts.Node
		}

		results = append(results, NodeMetricResult{
			Node:      node,
			Value:     float64(sample.Value),
			Timestamp: sample.Timestamp.Time(),
		})
	}

	return results, warnings, nil
}

// Queries namespace CPU usage metrics (aggregated)
func QueryNamespaceCPU(ctx context.Context, opts QueryOptions) ([]NamespaceMetricResult, v1.Warnings, error) {
	query := BuildNamespaceCPUQuery(opts.Namespace, opts.RangeDuration)

	result, warnings, err := QueryWithCache(ctx, query, opts)
	if err != nil {
		return nil, warnings, err
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a vector, got %T", result)
	}

	results := make([]NamespaceMetricResult, 0, len(vector))
	for _, sample := range vector {
		namespace := string(sample.Metric["namespace"])

		results = append(results, NamespaceMetricResult{
			Namespace: namespace,
			Value:     float64(sample.Value),
			Timestamp: sample.Timestamp.Time(),
		})
	}

	return results, warnings, nil
}

// Queries namespace memory usage metrics (aggregated)
func QueryNamespaceMemory(ctx context.Context, opts QueryOptions) ([]NamespaceMetricResult, v1.Warnings, error) {
	query := BuildNamespaceMemoryQuery(opts.Namespace)

	result, warnings, err := QueryWithCache(ctx, query, opts)
	if err != nil {
		return nil, warnings, err
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a vector, got %T", result)
	}

	results := make([]NamespaceMetricResult, 0, len(vector))
	for _, sample := range vector {
		namespace := string(sample.Metric["namespace"])

		results = append(results, NamespaceMetricResult{
			Namespace: namespace,
			Value:     float64(sample.Value),
			Timestamp: sample.Timestamp.Time(),
		})
	}

	return results, warnings, nil
}

// Queries pod CPU usage metrics over a time range
func QueryPodCPURange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildPodCPUQuery(opts.Namespace, opts.Pod, opts.Container, opts.RangeDuration)

	result, warnings, err := QueryRangeWithCache(ctx, query, r, opts)
	if err != nil {
		return nil, warnings, err
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a matrix, got %T", result)
	}

	return matrix, warnings, nil
}

// Queries pod memory usage metrics over a time range
func QueryPodMemoryRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildPodMemoryQuery(opts.Namespace, opts.Pod, opts.Container)

	result, warnings, err := QueryRangeWithCache(ctx, query, r, opts)
	if err != nil {
		return nil, warnings, err
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a matrix, got %T", result)
	}

	return matrix, warnings, nil
}

// Queries node CPU usage metrics over a time range
func QueryNodeCPURange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildNodeCPUQuery(opts.Node, opts.RangeDuration)

	result, warnings, err := QueryRangeWithCache(ctx, query, r, opts)
	if err != nil {
		return nil, warnings, err
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a matrix, got %T", result)
	}

	return matrix, warnings, nil
}

// Queries node memory usage metrics over a time range
func QueryNodeMemoryRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildNodeMemoryQuery(opts.Node)

	result, warnings, err := QueryRangeWithCache(ctx, query, r, opts)
	if err != nil {
		return nil, warnings, err
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a matrix, got %T", result)
	}

	return matrix, warnings, nil
}

// Queries namespace CPU usage metrics over a time range (aggregated)
func QueryNamespaceCPURange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildNamespaceCPUQuery(opts.Namespace, opts.RangeDuration)

	result, warnings, err := QueryRangeWithCache(ctx, query, r, opts)
	if err != nil {
		return nil, warnings, err
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a matrix, got %T", result)
	}

	return matrix, warnings, nil
}

// Queries namespace memory usage metrics over a time range (aggregated)
func QueryNamespaceMemoryRange(ctx context.Context, r v1.Range, opts QueryOptions) (model.Matrix, v1.Warnings, error) {
	query := BuildNamespaceMemoryQuery(opts.Namespace)

	result, warnings, err := QueryRangeWithCache(ctx, query, r, opts)
	if err != nil {
		return nil, warnings, err
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, warnings, fmt.Errorf("query result is not a matrix, got %T", result)
	}

	return matrix, warnings, nil
}
