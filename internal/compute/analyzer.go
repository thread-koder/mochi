package compute

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/thread_koder/mochi/internal/database"
	"github.com/thread_koder/mochi/internal/prometheus"
	"github.com/thread_koder/mochi/internal/timeseries"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Holds configuration for analysis
type AnalysisOptions struct {
	TimeRange         time.Duration // How far back to analyze (default: 24h)
	RangeStep         time.Duration // Step size for range queries (default: 1m)
	IncludeTimeSeries bool          // Whether to include raw datapoints for charting
}

// Returns default analysis options
func DefaultAnalysisOptions() AnalysisOptions {
	opts := AnalysisOptions{
		TimeRange:         24 * time.Hour,
		RangeStep:         1 * time.Minute,
		IncludeTimeSeries: false,
	}
	opts.SetTimeRange(opts.TimeRange)
	return opts
}

// Sets the time range and adjusts the step size to respect Prometheus limits
func (opts *AnalysisOptions) SetTimeRange(timeRange time.Duration) {
	opts.TimeRange = timeRange
	const maxPoints = 11000

	// Calculate minimum step needed
	totalMinutes := timeRange.Minutes()
	minStepMinutes := totalMinutes / maxPoints

	// Round up to next reasonable interval
	if minStepMinutes <= 1 {
		opts.RangeStep = 1 * time.Minute
	} else if minStepMinutes <= 5 {
		opts.RangeStep = 5 * time.Minute
	} else if minStepMinutes <= 15 {
		opts.RangeStep = 15 * time.Minute
	} else if minStepMinutes <= 30 {
		opts.RangeStep = 30 * time.Minute
	} else if minStepMinutes <= 60 {
		opts.RangeStep = 1 * time.Hour
	} else if minStepMinutes <= 240 {
		opts.RangeStep = 4 * time.Hour
	} else {
		// For very long ranges, use 6-hour steps
		opts.RangeStep = 6 * time.Hour
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
	ContainerName string             `json:"container_name"`
	Utilization   UtilizationResult  `json:"utilization"`
	Provisioning  ProvisioningResult `json:"provisioning"`
	Stability     StabilityResult    `json:"stability"`
	TimeSeries    *TimeSeries        `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

// Represents analysis results for a pod
type PodAnalysis struct {
	PodUID      string              `json:"pod_uid"`
	PodName     string              `json:"pod_name"`
	Containers  []ContainerAnalysis `json:"containers"`            // Individual container analyses
	Utilization UtilizationResult   `json:"utilization"`           // Aggregated from containers
	Stability   StabilityResult     `json:"stability"`             // Aggregated from containers
	TimeSeries  *TimeSeries         `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

// Represents analysis results for a workload
type WorkloadAnalysis struct {
	WorkloadType string            `json:"workload_type"`
	WorkloadName string            `json:"workload_name"`
	Namespace    string            `json:"namespace"`
	Pods         []PodAnalysis     `json:"pods"` // Individual pod analyses
	Utilization  UtilizationResult `json:"utilization"`
	Stability    StabilityResult   `json:"stability"`             // Aggregated from pods
	TimeSeries   *TimeSeries       `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

// Represents analysis results for a namespace
type NamespaceAnalysis struct {
	Namespace   string             `json:"namespace"`
	Utilization UtilizationResult  `json:"utilization"`           // Aggregated from all workloads/pods
	Stability   StabilityResult    `json:"stability"`             // Aggregated from all workloads/pods
	Workloads   []WorkloadAnalysis `json:"workloads,omitempty"`   // Optional: individual workload analyses
	TimeSeries  *TimeSeries        `json:"time_series,omitempty"` // Optional: raw datapoints for charting
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

	// Analyze stability
	stability, err := AnalyzeStability(metrics)
	if err != nil {
		return ContainerAnalysis{}, fmt.Errorf("failed to analyze stability: %w", err)
	}

	// Parse resource specs
	specs, err := ParseContainerSpecs(container)
	if err != nil {
		return ContainerAnalysis{}, fmt.Errorf("failed to parse container specs: %w", err)
	}

	// Analyze provisioning
	provisioning, err := AnalyzeProvisioning(specs, utilization, stability)
	if err != nil {
		return ContainerAnalysis{}, fmt.Errorf("failed to analyze provisioning: %w", err)
	}

	result := ContainerAnalysis{
		ContainerName: container.Name,
		Utilization:   utilization,
		Stability:     stability,
		Provisioning:  provisioning,
	}

	// Include time series if requested
	if opts.IncludeTimeSeries {
		result.TimeSeries = &TimeSeries{
			CPU:    metrics.CPU,
			Memory: metrics.Memory,
		}
	}

	return result, nil
}

// Analyzes a pod and its containers
func AnalyzePod(ctx context.Context, pod *database.Pod, containers []*database.Container, opts AnalysisOptions) (PodAnalysis, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return PodAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	// Analyze containers in parallel
	containerAnalyses := make([]ContainerAnalysis, len(containers))
	stabilities := make([]StabilityResult, len(containers))
	g, gctx := errgroup.WithContext(ctx)
	for i, container := range containers {
		g.Go(func() error {
			analysis, err := AnalyzeContainer(gctx, container, opts)
			if err != nil {
				return fmt.Errorf("failed to analyze container %s: %w", container.Name, err)
			}
			containerAnalyses[i] = analysis
			stabilities[i] = analysis.Stability
			return nil
		})
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

	// Wait for all containers to be analyzed and check for errors
	if err := g.Wait(); err != nil {
		return PodAnalysis{}, err
	}

	result := PodAnalysis{
		PodUID:      pod.UID,
		PodName:     pod.Name,
		Containers:  containerAnalyses,
		Utilization: utilization,
		Stability:   AggregateStability(stabilities),
	}

	// Include time series if requested
	if opts.IncludeTimeSeries {
		result.TimeSeries = &TimeSeries{
			CPU:    metrics.CPU,
			Memory: metrics.Memory,
		}
	}

	return result, nil
}

// Analyzes a workload and its pods
func AnalyzeWorkload(ctx context.Context, workloadType string, workloadName string, namespace string, pods []*database.Pod, opts AnalysisOptions) (WorkloadAnalysis, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	// Validate inputs
	if len(pods) == 0 {
		return WorkloadAnalysis{}, fmt.Errorf("no pods found for workload %s/%s", namespace, workloadName)
	}

	// Disable time series for pods
	podOpts := opts
	podOpts.IncludeTimeSeries = false

	// Analyze pods in parallel
	podAnalyses := make([]PodAnalysis, len(pods))
	stabilities := make([]StabilityResult, len(pods))
	g, gctx := errgroup.WithContext(ctx)
	for i, pod := range pods {
		g.Go(func() error {
			containers, err := database.GetContainersByPodUID(gctx, pod.UID)
			if err != nil {
				return fmt.Errorf("failed to fetch containers for pod %s: %w", pod.Name, err)
			}
			podAnalysis, err := AnalyzePod(gctx, pod, containers, podOpts)
			if err != nil {
				return fmt.Errorf("failed to analyze pod %s: %w", pod.Name, err)
			}
			podAnalyses[i] = podAnalysis
			stabilities[i] = podAnalysis.Stability
			return nil
		})
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

	// Wait for all pods to be analyzed and check for errors
	if err := g.Wait(); err != nil {
		return WorkloadAnalysis{}, err
	}

	result := WorkloadAnalysis{
		WorkloadType: workloadType,
		WorkloadName: workloadName,
		Namespace:    namespace,
		Pods:         podAnalyses,
		Utilization:  utilization,
		Stability:    AggregateStability(stabilities),
	}

	// Include time series if requested
	if opts.IncludeTimeSeries {
		result.TimeSeries = &TimeSeries{
			CPU:    metrics.CPU,
			Memory: metrics.Memory,
		}
	}

	return result, nil
}

// Analyzes a namespace
func AnalyzeNamespace(ctx context.Context, namespace string, opts AnalysisOptions, includeWorkloads bool) (NamespaceAnalysis, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	// Disable time series for workloads
	workloadOpts := opts
	workloadOpts.IncludeTimeSeries = false

	// Analyze workloads in parallel
	var workloads []WorkloadAnalysis
	var stability StabilityResult
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		workloadsAnalyses, err := analyzeNamespaceWorkloads(gctx, namespace, workloadOpts)
		if err != nil {
			return fmt.Errorf("failed to analyze namespace workloads: %w", err)
		}
		workloads = workloadsAnalyses
		stabilities := make([]StabilityResult, 0, len(workloads))
		for _, w := range workloads {
			stabilities = append(stabilities, w.Stability)
		}
		stability = AggregateStability(stabilities)
		return nil
	})

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

	// Wait for all workloads to be analyzed and check for errors
	if err := g.Wait(); err != nil {
		return NamespaceAnalysis{}, err
	}

	result := NamespaceAnalysis{
		Namespace:   namespace,
		Utilization: utilization,
		Stability:   stability,
	}

	// Include workloads if requested
	if includeWorkloads {
		result.Workloads = workloads
	}

	// Include time series if requested
	if opts.IncludeTimeSeries {
		result.TimeSeries = &TimeSeries{
			CPU:    metrics.CPU,
			Memory: metrics.Memory,
		}
	}

	return result, nil
}

// Analyzes all workloads in a namespace
func analyzeNamespaceWorkloads(ctx context.Context, namespace string, opts AnalysisOptions) ([]WorkloadAnalysis, error) {
	workloads := make([]WorkloadAnalysis, 0)
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(ctx)

	// A worker function to analyze a workload
	analyzeSingle := func(kind, name string, pods []*database.Pod) {
		g.Go(func() error {
			if len(pods) == 0 {
				return nil // skip
			}
			analysis, err := AnalyzeWorkload(gctx, kind, name, namespace, pods, opts)
			if err != nil {
				return fmt.Errorf("failed to analyze workload %s/%s: %w", kind, name, err)
			}

			mu.Lock()
			workloads = append(workloads, analysis)
			mu.Unlock()
			return nil
		})
	}

	// A worker function to fetch the pods for a workload and analyze it
	fetchAndAnalyze := func(kind, name string) {
		g.Go(func() error {
			pods, err := database.GetPodsByWorkload(gctx, kind, name, namespace)
			if err != nil {
				return fmt.Errorf("failed to fetch pods for workload %s/%s: %w", kind, name, err)
			}
			if len(pods) > 0 {
				analyzeSingle(kind, name, pods)
			}
			return nil
		})
	}

	g.Go(func() error {
		deps, err := database.GetDeploymentsByNamespace(gctx, namespace)
		if err != nil {
			return fmt.Errorf("failed to fetch deployments for namespace %s: %w", namespace, err)
		}
		for _, d := range deps {
			fetchAndAnalyze("Deployment", d.Name)
		}
		return nil
	})

	g.Go(func() error {
		stss, err := database.GetStatefulSetsByNamespace(gctx, namespace)
		if err != nil {
			return fmt.Errorf("failed to fetch statefulsets for namespace %s: %w", namespace, err)
		}
		for _, s := range stss {
			fetchAndAnalyze("StatefulSet", s.Name)
		}
		return nil
	})

	g.Go(func() error {
		dss, err := database.GetDaemonSetsByNamespace(gctx, namespace)
		if err != nil {
			return fmt.Errorf("failed to fetch daemonsets for namespace %s: %w", namespace, err)
		}
		for _, ds := range dss {
			fetchAndAnalyze("DaemonSet", ds.Name)
		}
		return nil
	})

	g.Go(func() error {
		pods, err := database.GetStandalonePodsByNamespace(gctx, namespace)
		if err != nil {
			return fmt.Errorf("failed to fetch standalone pods for namespace %s: %w", namespace, err)
		}
		for _, p := range pods {
			analyzeSingle("Pod", p.Name, []*database.Pod{p})
		}
		return nil
	})

	g.Go(func() error {
		pods, err := database.GetPodsByOwnerKind(gctx, "Node", namespace)
		if err != nil {
			return fmt.Errorf("failed to fetch system pods for namespace %s: %w", namespace, err)
		}
		for _, p := range pods {
			analyzeSingle("Pod", p.Name, []*database.Pod{p})
		}
		return nil
	})

	// Wait for all queries to complete and check for errors
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return workloads, nil
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

	queryOpts := prometheus.QueryOptions{
		Namespace:     container.Namespace,
		Pod:           container.PodName,
		Container:     container.Name,
		RangeDuration: "5m",
	}

	var (
		cpuMatrix     model.Matrix
		memoryMatrix  model.Matrix
		cpuThrottling float64
		cpuPressure   float64
		memFailCnt    float64
		memOOM        float64
		memPressure   float64
		restarts      float64
	)

	// Execute all queries in parallel
	g, gctx := errgroup.WithContext(ctx)

	// Query CPU metrics
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodCPURange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU metrics: %w", err)
		}
		cpuMatrix = matrix
		return nil
	})

	// Query memory metrics
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodMemoryRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory metrics: %w", err)
		}
		memoryMatrix = matrix
		return nil
	})

	// Query CPU throttling metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodCPUThrottling(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU throttling metrics: %w", err)
		}
		cpuThrottling = value
		return nil
	})

	// Query CPU pressure metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodCPUPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU pressure metrics: %w", err)
		}
		cpuPressure = value
		return nil
	})

	// Query memory fail count metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryFailCount(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory fail count metrics: %w", err)
		}
		memFailCnt = value
		return nil
	})

	// Query memory OOM metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryOOM(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory OOM metrics: %w", err)
		}
		memOOM = value
		return nil
	})

	// Query memory pressure metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory pressure metrics: %w", err)
		}
		memPressure = value
		return nil
	})

	// Query container restarts metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryContainerRestarts(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query container restarts metrics: %w", err)
		}
		restarts = value
		return nil
	})

	// Wait for all queries to complete and check for errors
	if err := g.Wait(); err != nil {
		return ResourceMetrics{}, err
	}

	return ResourceMetrics{
		CPU:            timeseries.MatrixToDataPoints(cpuMatrix),
		Memory:         timeseries.MatrixToDataPoints(memoryMatrix),
		CPUThrottling:  []timeseries.DataPoint{{Timestamp: time.Now(), Value: cpuThrottling}},
		CPUPressure:    []timeseries.DataPoint{{Timestamp: time.Now(), Value: cpuPressure}},
		MemoryFailCnt:  []timeseries.DataPoint{{Timestamp: time.Now(), Value: memFailCnt}},
		MemoryOOM:      []timeseries.DataPoint{{Timestamp: time.Now(), Value: memOOM}},
		MemoryPressure: []timeseries.DataPoint{{Timestamp: time.Now(), Value: memPressure}},
		Restarts:       []timeseries.DataPoint{{Timestamp: time.Now(), Value: restarts}},
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

	queryOpts := prometheus.QueryOptions{
		Namespace:     pod.Namespace,
		Pod:           pod.Name,
		RangeDuration: "5m",
	}

	var cpuMatrix, memoryMatrix model.Matrix

	// Execute queries in parallel
	g, gctx := errgroup.WithContext(ctx)

	// Query pod CPU metrics
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodCPURange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query pod CPU metrics: %w", err)
		}
		cpuMatrix = matrix
		return nil
	})

	// Query pod memory metrics
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodMemoryRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query pod memory metrics: %w", err)
		}
		memoryMatrix = matrix
		return nil
	})

	// Wait for all queries to complete and check for errors
	if err := g.Wait(); err != nil {
		return ResourceMetrics{}, err
	}

	// Aggregate CPU metrics
	cpuDataPoints := timeseries.MatrixToDataPoints(cpuMatrix)
	aggregatedCPU := timeseries.AggregateDataPointsByTimestamp(cpuDataPoints)

	// Aggregate memory metrics
	memoryDataPoints := timeseries.MatrixToDataPoints(memoryMatrix)
	aggregatedMemory := timeseries.AggregateDataPointsByTimestamp(memoryDataPoints)

	return ResourceMetrics{
		CPU:    aggregatedCPU,
		Memory: aggregatedMemory,
	}, nil
}

// Aggregates metrics from all pods in a workload
func fetchWorkloadMetrics(ctx context.Context, pods []*database.Pod, opts AnalysisOptions) (ResourceMetrics, error) {
	if len(pods) == 0 {
		return ResourceMetrics{}, fmt.Errorf("no pods found for workload")
	}

	// Set up time range
	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	// Per-pod results: each goroutine writes to its index
	type podMetrics struct {
		CPU    []timeseries.DataPoint
		Memory []timeseries.DataPoint
	}
	results := make([]podMetrics, len(pods))

	// Query all pods in parallel
	g, gctx := errgroup.WithContext(ctx)

	for i, pod := range pods {
		queryOpts := prometheus.QueryOptions{
			Namespace:     pod.Namespace,
			Pod:           pod.Name,
			RangeDuration: "5m",
		}

		g.Go(func() error {
			// Query CPU and Memory metrics in parallel for this pod
			var cpuMatrix, memoryMatrix model.Matrix
			// Create a new error group for this pod
			podG, podCtx := errgroup.WithContext(gctx)

			// Query CPU metrics
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodCPURange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query CPU metrics for pod %s/%s: %w", pod.Namespace, pod.Name, err)
				}
				cpuMatrix = matrix
				return nil
			})

			// Query memory metrics
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodMemoryRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query memory metrics for pod %s/%s: %w", pod.Namespace, pod.Name, err)
				}
				memoryMatrix = matrix
				return nil
			})

			// Wait for both queries to complete and check for errors for this pod
			if err := podG.Wait(); err != nil {
				return err
			}

			results[i] = podMetrics{
				CPU:    timeseries.MatrixToDataPoints(cpuMatrix),
				Memory: timeseries.MatrixToDataPoints(memoryMatrix),
			}
			return nil
		})
	}

	// Wait for all queries to complete and check for errors
	if err := g.Wait(); err != nil {
		return ResourceMetrics{}, err
	}

	// Merge all per-pod results
	var cpu, memory []timeseries.DataPoint
	for _, p := range results {
		cpu = timeseries.MergeDataPointsByTime(cpu, p.CPU)
		memory = timeseries.MergeDataPointsByTime(memory, p.Memory)
	}

	return ResourceMetrics{
		CPU:    cpu,
		Memory: memory,
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

	queryOpts := prometheus.QueryOptions{
		Namespace:     namespace,
		RangeDuration: "5m",
	}

	var cpuMatrix, memoryMatrix model.Matrix

	// Execute queries in parallel
	g, gctx := errgroup.WithContext(ctx)

	// Query namespace CPU metrics
	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceCPURange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query namespace CPU metrics: %w", err)
		}
		cpuMatrix = matrix
		return nil
	})

	// Query namespace memory metrics
	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceMemoryRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query namespace memory metrics: %w", err)
		}
		memoryMatrix = matrix
		return nil
	})

	// Wait for all queries to complete and check for errors
	if err := g.Wait(); err != nil {
		return ResourceMetrics{}, err
	}

	return ResourceMetrics{
		CPU:    timeseries.MatrixToDataPoints(cpuMatrix),
		Memory: timeseries.MatrixToDataPoints(memoryMatrix),
	}, nil
}

// Parses resource specs from database container model
func ParseContainerSpecs(container *database.Container) (ResourceSpecs, error) {
	specs := ResourceSpecs{}

	// Parse CPU request
	if container.CPURequest != nil && *container.CPURequest != "" {
		qty, err := resource.ParseQuantity(*container.CPURequest)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse CPU request: %w", err)
		}
		specs.CPURequest = new(qty.AsFloat64Slow())
	}

	// Parse CPU limit
	if container.CPULimit != nil && *container.CPULimit != "" {
		qty, err := resource.ParseQuantity(*container.CPULimit)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse CPU limit: %w", err)
		}
		specs.CPULimit = new(qty.AsFloat64Slow())
	}

	// Parse memory request
	if container.MemoryRequest != nil && *container.MemoryRequest != "" {
		qty, err := resource.ParseQuantity(*container.MemoryRequest)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse memory request: %w", err)
		}
		specs.MemoryRequest = new(float64(qty.Value()))
	}

	// Parse memory limit
	if container.MemoryLimit != nil && *container.MemoryLimit != "" {
		qty, err := resource.ParseQuantity(*container.MemoryLimit)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse memory limit: %w", err)
		}
		specs.MemoryLimit = new(float64(qty.Value()))
	}

	return specs, nil
}
