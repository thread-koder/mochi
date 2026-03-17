package compute

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/thread_koder/mochi/internal/database"
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
	Containers  []ContainerAnalysis `json:"containers"`
	Utilization UtilizationResult   `json:"utilization"`
	Stability   StabilityResult     `json:"stability"`
	TimeSeries  *TimeSeries         `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

// Represents analysis results for a workload
type WorkloadAnalysis struct {
	WorkloadType string            `json:"workload_type"`
	WorkloadName string            `json:"workload_name"`
	Namespace    string            `json:"namespace"`
	Pods         []PodAnalysis     `json:"pods"`
	Utilization  UtilizationResult `json:"utilization"`
	Stability    StabilityResult   `json:"stability"`
	TimeSeries   *TimeSeries       `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

// Represents analysis results for a namespace
type NamespaceAnalysis struct {
	Namespace   string             `json:"namespace"`
	Utilization UtilizationResult  `json:"utilization"`
	Stability   StabilityResult    `json:"stability"`
	Workloads   []WorkloadAnalysis `json:"workloads"`
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

	// Analyze the pod's containers in parallel
	containerAnalyses := make([]ContainerAnalysis, len(containers))
	g, gctx := errgroup.WithContext(ctx)
	for i, container := range containers {
		g.Go(func() error {
			analysis, err := AnalyzeContainer(gctx, container, opts)
			if err != nil {
				return fmt.Errorf("failed to analyze container %s: %w", container.Name, err)
			}
			containerAnalyses[i] = analysis
			return nil
		})
	}

	// Fetch pod-level metrics
	metrics, err := fetchPodMetrics(ctx, pod, opts)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to fetch pod metrics: %w", err)
	}

	// Validate we have at least some metrics
	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return PodAnalysis{}, fmt.Errorf("no metrics available for pod %s", pod.Name)
	}

	// Analyze utilization
	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to analyze pod utilization: %w", err)
	}

	// Analyze stability
	stability, err := AnalyzeStability(metrics)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to analyze pod stability: %w", err)
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
		Stability:   stability,
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
func AnalyzeWorkload(ctx context.Context, workloadType string, workloadName string, namespace string, pods []*database.Pod, opts AnalysisOptions, includePods bool) (WorkloadAnalysis, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	// Validate inputs
	if len(pods) == 0 {
		return WorkloadAnalysis{}, fmt.Errorf("no pods found for workload %s/%s", namespace, workloadName)
	}

	// Analyze pods in parallel if requested
	var podAnalyses []PodAnalysis
	g, gctx := errgroup.WithContext(ctx)
	if includePods {
		podAnalyses = make([]PodAnalysis, len(pods))
		// Disable time series for pods
		podOpts := opts
		podOpts.IncludeTimeSeries = false
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
				return nil
			})
		}
	}

	// Fetch workload-level metrics
	metrics, err := fetchWorkloadMetrics(ctx, pods, opts)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("failed to fetch workload metrics: %w", err)
	}

	// Validate we have at least some metrics
	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return WorkloadAnalysis{}, fmt.Errorf("no metrics available for workload %s/%s", namespace, workloadName)
	}

	// Analyze utilization
	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("failed to analyze workload utilization: %w", err)
	}

	// Analyze stability
	stability, err := AnalyzeStability(metrics)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("failed to analyze workload stability: %w", err)
	}

	// Wait for all pods to be analyzed if requested and check for errors
	if err := g.Wait(); err != nil {
		return WorkloadAnalysis{}, err
	}

	result := WorkloadAnalysis{
		WorkloadType: workloadType,
		WorkloadName: workloadName,
		Namespace:    namespace,
		Pods:         podAnalyses,
		Utilization:  utilization,
		Stability:    stability,
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
func AnalyzeNamespace(ctx context.Context, namespace string, opts AnalysisOptions) (NamespaceAnalysis, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	// Disable time series for workloads
	workloadOpts := opts
	workloadOpts.IncludeTimeSeries = false

	// Analyze workloads in parallel
	var workloads []WorkloadAnalysis
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		workloadsAnalyses, err := analyzeNamespaceWorkloads(gctx, namespace, workloadOpts)
		if err != nil {
			return fmt.Errorf("failed to analyze namespace workloads: %w", err)
		}
		workloads = workloadsAnalyses
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

	// Analyze utilization
	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("failed to analyze namespace utilization: %w", err)
	}

	// Analyze stability
	stability, err := AnalyzeStability(metrics)
	if err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("failed to analyze namespace stability: %w", err)
	}

	// Wait for all workloads to be analyzed and check for errors
	if err := g.Wait(); err != nil {
		return NamespaceAnalysis{}, err
	}

	result := NamespaceAnalysis{
		Namespace:   namespace,
		Utilization: utilization,
		Stability:   stability,
		Workloads:   workloads,
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
			// Analyze the workload without including its pods analyses
			analysis, err := AnalyzeWorkload(gctx, kind, name, namespace, pods, opts, false)
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

	// Wait for all workloads to be analyzed and check for errors
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return workloads, nil
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
