package analyzer

import (
	"context"
	"fmt"
	"sort"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/thread_koder/mochi/internal/database"
	"github.com/thread_koder/mochi/internal/logger"
	"github.com/thread_koder/mochi/internal/prometheus"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Holds configuration for analysis
type AnalysisOptions struct {
	TimeRange time.Duration // How far back to analyze (default: 24h)
	RangeStep time.Duration // Step size for range queries (default: 1m)
	UseCache  bool          // Whether to use Prometheus query cache
	CacheTTL  time.Duration // Cache TTL if UseCache is true
}

// Returns default analysis options
func DefaultAnalysisOptions() AnalysisOptions {
	return AnalysisOptions{
		TimeRange: 24 * time.Hour,
		RangeStep: 1 * time.Minute,
		UseCache:  true,
		CacheTTL:  5 * time.Minute,
	}
}

// Validates analysis options
func (opts AnalysisOptions) Validate() error {
	if opts.TimeRange <= 0 {
		return fmt.Errorf("TimeRange must be positive, got: %v", opts.TimeRange)
	}
	if opts.RangeStep <= 0 {
		return fmt.Errorf("RangeStep must be positive, got: %v", opts.RangeStep)
	}
	return nil
}

// Represents analysis results for a container
type ContainerAnalysis struct {
	ContainerName string
	Utilization   UtilizationResult
	Provisioning  ProvisioningResult
}

// Represents analysis results for a pod
type PodAnalysis struct {
	PodUID      string
	PodName     string
	Containers  []ContainerAnalysis // Individual container analyses
	Utilization UtilizationResult   // Aggregated from containers
}

// Represents analysis results for a workload
type WorkloadAnalysis struct {
	WorkloadType string
	WorkloadName string
	Namespace    string
	Pods         []PodAnalysis // Individual pod analyses
	Utilization  UtilizationResult
}

// Represents analysis results for a namespace
type NamespaceAnalysis struct {
	Namespace   string
	Utilization UtilizationResult // Aggregated from all workloads/pods
}

// Analyzes a single container's resource utilization and provisioning
func AnalyzeContainer(ctx context.Context, container *database.Container, opts AnalysisOptions) (ContainerAnalysis, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return ContainerAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	if container == nil {
		return ContainerAnalysis{}, fmt.Errorf("container cannot be nil")
	}

	// Fetch metrics from Prometheus
	metrics, err := fetchContainerMetrics(ctx, container, opts)
	if err != nil {
		return ContainerAnalysis{}, fmt.Errorf("failed to fetch container metrics: %w", err)
	}

	// Validate we have some metrics
	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return ContainerAnalysis{}, fmt.Errorf("no metrics available for container %s", container.Name)
	}

	// Analyze utilization
	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return ContainerAnalysis{}, fmt.Errorf("failed to analyze utilization: %w", err)
	}

	// Parse resource specs
	specs, err := parseContainerSpecs(container)
	if err != nil {
		return ContainerAnalysis{}, fmt.Errorf("failed to parse container specs: %w", err)
	}

	// Analyze provisioning
	provisioning, err := AnalyzeProvisioning(specs, utilization)
	if err != nil {
		return ContainerAnalysis{}, fmt.Errorf("failed to analyze provisioning: %w", err)
	}

	return ContainerAnalysis{
		ContainerName: container.Name,
		Utilization:   utilization,
		Provisioning:  provisioning,
	}, nil
}

// Analyzes a pod and its containers
func AnalyzePod(ctx context.Context, pod *database.Pod, containers []*database.Container, opts AnalysisOptions) (PodAnalysis, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return PodAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	// Analyze each container individually
	log := logger.WithComponent("analyzer")
	containerAnalyses := make([]ContainerAnalysis, 0, len(containers))
	for _, container := range containers {
		analysis, err := AnalyzeContainer(ctx, container, opts)
		if err != nil {
			log.Warn().
				Err(err).
				Str("container", container.Name).
				Str("pod", pod.Name).
				Msg("Failed to analyze container, skipping")
			continue
		}
		containerAnalyses = append(containerAnalyses, analysis)
	}

	// Aggregate metrics from all containers for pod-level utilization
	metrics, err := fetchPodMetrics(ctx, pod, opts)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to fetch pod metrics: %w", err)
	}

	// Validate we have at least some metrics
	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return PodAnalysis{}, fmt.Errorf("no metrics available for pod %s", pod.Name)
	}

	// Analyze aggregated utilization
	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to analyze pod utilization: %w", err)
	}

	return PodAnalysis{
		PodUID:      pod.UID,
		PodName:     pod.Name,
		Containers:  containerAnalyses,
		Utilization: utilization,
	}, nil
}

// Analyzes a workload and its pods
func AnalyzeWorkload(ctx context.Context, workloadType string, workloadName string, namespace string, pods []*database.Pod, opts AnalysisOptions) (WorkloadAnalysis, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	// Validate inputs
	if len(pods) == 0 {
		return WorkloadAnalysis{}, fmt.Errorf("no pods provided for workload %s/%s", namespace, workloadName)
	}

	// Analyze each pod individually
	log := logger.WithComponent("analyzer")
	podAnalyses := make([]PodAnalysis, 0, len(pods))
	for _, pod := range pods {
		// Fetch containers for this pod
		containers, err := database.GetContainersByPodUID(ctx, pod.UID)
		if err != nil {
			log.Warn().
				Err(err).
				Str("pod", pod.Name).
				Str("workload", workloadName).
				Msg("Failed to fetch containers for pod, skipping")
			continue
		}

		// Analyze the pod
		podAnalysis, err := AnalyzePod(ctx, pod, containers, opts)
		if err != nil {
			log.Warn().
				Err(err).
				Str("pod", pod.Name).
				Str("workload", workloadName).
				Msg("Failed to analyze pod, skipping")
			continue
		}
		podAnalyses = append(podAnalyses, podAnalysis)
	}

	// Aggregate metrics from all pods for workload-level utilization
	metrics, err := fetchWorkloadMetrics(ctx, pods, opts)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("failed to fetch workload metrics: %w", err)
	}

	// Validate we have at least some metrics
	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return WorkloadAnalysis{}, fmt.Errorf("no metrics available for workload %s/%s", namespace, workloadName)
	}

	// Analyze aggregated utilization
	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("failed to analyze workload utilization: %w", err)
	}

	return WorkloadAnalysis{
		WorkloadType: workloadType,
		WorkloadName: workloadName,
		Namespace:    namespace,
		Pods:         podAnalyses,
		Utilization:  utilization,
	}, nil
}

// Analyzes a namespace
func AnalyzeNamespace(ctx context.Context, namespace string, opts AnalysisOptions) (NamespaceAnalysis, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	// Fetch namespace-level metrics
	metrics, err := fetchNamespaceMetrics(ctx, namespace, opts)
	if err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("failed to fetch namespace metrics: %w", err)
	}

	// Validate we have some metrics
	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return NamespaceAnalysis{}, fmt.Errorf("no metrics available for namespace %s", namespace)
	}

	// Analyze aggregated utilization
	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("failed to analyze namespace utilization: %w", err)
	}

	return NamespaceAnalysis{
		Namespace:   namespace,
		Utilization: utilization,
	}, nil
}

// Fetches container metrics
func fetchContainerMetrics(ctx context.Context, container *database.Container, opts AnalysisOptions) (ResourceMetrics, error) {
	// Set up time range
	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	// Query CPU metrics
	cpuMatrix, _, err := prometheus.QueryPodCPURange(ctx, r, prometheus.QueryOptions{
		Namespace:     container.Namespace,
		Pod:           container.PodName,
		Container:     container.Name,
		UseCache:      opts.UseCache,
		CacheTTL:      opts.CacheTTL,
		RangeDuration: opts.TimeRange.String(),
	})
	if err != nil {
		return ResourceMetrics{}, fmt.Errorf("failed to query CPU metrics: %w", err)
	}

	// Query memory metrics
	memoryMatrix, _, err := prometheus.QueryPodMemoryRange(ctx, r, prometheus.QueryOptions{
		Namespace: container.Namespace,
		Pod:       container.PodName,
		Container: container.Name,
		UseCache:  opts.UseCache,
		CacheTTL:  opts.CacheTTL,
	})
	if err != nil {
		return ResourceMetrics{}, fmt.Errorf("failed to query memory metrics: %w", err)
	}

	return ResourceMetrics{
		CPU:    MatrixToDataPoints(cpuMatrix),
		Memory: MatrixToDataPoints(memoryMatrix),
	}, nil
}

// Aggregates metrics from all containers in a pod
func fetchPodMetrics(ctx context.Context, pod *database.Pod, opts AnalysisOptions) (ResourceMetrics, error) {
	// Set up time range
	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	// Query CPU metrics for all containers in the pod
	cpuMatrix, _, err := prometheus.QueryPodCPURange(ctx, r, prometheus.QueryOptions{
		Namespace:     pod.Namespace,
		Pod:           pod.Name,
		UseCache:      opts.UseCache,
		CacheTTL:      opts.CacheTTL,
		RangeDuration: opts.TimeRange.String(),
	})
	if err != nil {
		return ResourceMetrics{}, fmt.Errorf("failed to query pod CPU metrics: %w", err)
	}

	cpuDataPoints := MatrixToDataPoints(cpuMatrix)
	aggregatedCPU := aggregateDataPointsByTimestamp(cpuDataPoints)

	// Query memory metrics for all containers in the pod
	memoryMatrix, _, err := prometheus.QueryPodMemoryRange(ctx, r, prometheus.QueryOptions{
		Namespace: pod.Namespace,
		Pod:       pod.Name,
		UseCache:  opts.UseCache,
		CacheTTL:  opts.CacheTTL,
	})
	if err != nil {
		return ResourceMetrics{}, fmt.Errorf("failed to query pod memory metrics: %w", err)
	}

	memoryDataPoints := MatrixToDataPoints(memoryMatrix)
	aggregatedMemory := aggregateDataPointsByTimestamp(memoryDataPoints)

	return ResourceMetrics{
		CPU:    aggregatedCPU,
		Memory: aggregatedMemory,
	}, nil
}

// Aggregates metrics from all pods in a workload
func fetchWorkloadMetrics(ctx context.Context, pods []*database.Pod, opts AnalysisOptions) (ResourceMetrics, error) {
	var aggregatedCPU, aggregatedMemory []DataPoint

	// Set up time range
	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	// Aggregate CPU metrics from all pods
	for _, pod := range pods {
		// Query all containers in the pod
		cpuMatrix, _, err := prometheus.QueryPodCPURange(ctx, r, prometheus.QueryOptions{
			Namespace:     pod.Namespace,
			Pod:           pod.Name,
			UseCache:      opts.UseCache,
			CacheTTL:      opts.CacheTTL,
			RangeDuration: opts.TimeRange.String(),
		})
		if err != nil {
			return ResourceMetrics{}, fmt.Errorf("failed to query pod CPU metrics: %w", err)
		}

		cpuDataPoints := MatrixToDataPoints(cpuMatrix)
		aggregatedCPU = mergeDataPointsByTime(aggregatedCPU, cpuDataPoints)
	}

	// Aggregate memory metrics from all pods
	for _, pod := range pods {
		memoryMatrix, _, err := prometheus.QueryPodMemoryRange(ctx, r, prometheus.QueryOptions{
			Namespace: pod.Namespace,
			Pod:       pod.Name,
			UseCache:  opts.UseCache,
			CacheTTL:  opts.CacheTTL,
		})
		if err != nil {
			return ResourceMetrics{}, fmt.Errorf("failed to query pod memory metrics: %w", err)
		}

		memoryDataPoints := MatrixToDataPoints(memoryMatrix)
		aggregatedMemory = mergeDataPointsByTime(aggregatedMemory, memoryDataPoints)
	}

	return ResourceMetrics{
		CPU:    aggregatedCPU,
		Memory: aggregatedMemory,
	}, nil
}

// Fetches namespace metrics
func fetchNamespaceMetrics(ctx context.Context, namespace string, opts AnalysisOptions) (ResourceMetrics, error) {
	// Set up time range
	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	// Query CPU metrics
	cpuMatrix, _, err := prometheus.QueryNamespaceCPURange(ctx, r, prometheus.QueryOptions{
		Namespace:     namespace,
		UseCache:      opts.UseCache,
		CacheTTL:      opts.CacheTTL,
		RangeDuration: opts.TimeRange.String(),
	})
	if err != nil {
		return ResourceMetrics{}, fmt.Errorf("failed to query namespace CPU metrics: %w", err)
	}

	// Query memory metrics
	memoryMatrix, _, err := prometheus.QueryNamespaceMemoryRange(ctx, r, prometheus.QueryOptions{
		Namespace: namespace,
		UseCache:  opts.UseCache,
		CacheTTL:  opts.CacheTTL,
	})
	if err != nil {
		return ResourceMetrics{}, fmt.Errorf("failed to query namespace memory metrics: %w", err)
	}

	return ResourceMetrics{
		CPU:    MatrixToDataPoints(cpuMatrix),
		Memory: MatrixToDataPoints(memoryMatrix),
	}, nil
}

// Parses resource specs from database container model
func parseContainerSpecs(container *database.Container) (ResourceSpecs, error) {
	specs := ResourceSpecs{}

	// Parse CPU request
	if container.CPURequest != nil && *container.CPURequest != "" {
		qty, err := resource.ParseQuantity(*container.CPURequest)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse CPU request: %w", err)
		}
		cpuCores := qty.AsFloat64Slow()
		specs.CPURequest = &cpuCores
	}

	// Parse CPU limit
	if container.CPULimit != nil && *container.CPULimit != "" {
		qty, err := resource.ParseQuantity(*container.CPULimit)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse CPU limit: %w", err)
		}
		cpuCores := qty.AsFloat64Slow()
		specs.CPULimit = &cpuCores
	}

	// Parse memory request
	if container.MemoryRequest != nil && *container.MemoryRequest != "" {
		qty, err := resource.ParseQuantity(*container.MemoryRequest)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse memory request: %w", err)
		}
		memoryBytes := float64(qty.Value())
		specs.MemoryRequest = &memoryBytes
	}

	// Parse memory limit
	if container.MemoryLimit != nil && *container.MemoryLimit != "" {
		qty, err := resource.ParseQuantity(*container.MemoryLimit)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse memory limit: %w", err)
		}
		memoryBytes := float64(qty.Value())
		specs.MemoryLimit = &memoryBytes
	}

	return specs, nil
}

// Merges two slices of data points by timestamp (sums values at the same timestamp)
// Used when combining data points from multiple sources (e.g., multiple pods in a workload)
func mergeDataPointsByTime(existing []DataPoint, new []DataPoint) []DataPoint {
	if len(existing) == 0 {
		// If existing is empty, still aggregate new in case it has duplicate timestamps
		return aggregateDataPointsByTimestamp(new)
	}

	// Create a map of timestamp -> sum of values
	timeMap := make(map[time.Time]float64)
	for _, dp := range existing {
		timeMap[dp.Timestamp] += dp.Value
	}
	for _, dp := range new {
		timeMap[dp.Timestamp] += dp.Value
	}

	return timeMapToSortedDataPoints(timeMap)
}

// Aggregates a single slice of data points by timestamp (sums values at the same timestamp)
// Used when a single data source contains multiple series (e.g., multiple containers in a pod)
func aggregateDataPointsByTimestamp(dataPoints []DataPoint) []DataPoint {
	if len(dataPoints) == 0 {
		return dataPoints
	}

	// Create a map of timestamp -> sum of values
	timeMap := make(map[time.Time]float64)
	for _, dp := range dataPoints {
		timeMap[dp.Timestamp] += dp.Value
	}

	return timeMapToSortedDataPoints(timeMap)
}

// Converts a timestamp->value map to a sorted slice of DataPoints
func timeMapToSortedDataPoints(timeMap map[time.Time]float64) []DataPoint {
	// Convert back to sorted slice
	result := make([]DataPoint, 0, len(timeMap))
	for ts, val := range timeMap {
		result = append(result, DataPoint{
			Timestamp: ts,
			Value:     val,
		})
	}

	// Sort by timestamp (O(n log n) instead of O(n²))
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}
