// Package compute loads pod, workload, and namespace CPU and memory metrics from Prometheus,
// derives utilization, stability, and provisioning, builds resource recommendations, and can apply
// them with Kubernetes server-side apply. Optional raw CPU and memory series are included when
// callers set AnalysisOptions.IncludeTimeSeries.
package compute

import (
	"context"
	"fmt"
	"time"

	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/analyzer"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/resource"
)

// AnalysisOptions controls the analysis window, Prometheus range-query resolution, and whether
// results include raw CPU and memory series for charts.
type AnalysisOptions struct {
	TimeRange         time.Duration // How far back to analyze (default: 24h).
	RangeStep         time.Duration // Step size for Prometheus range queries (default: 1m).
	IncludeTimeSeries bool          // Whether to include raw CPU and memory series for charts (default: false).
}

// DefaultAnalysisOptions returns a 24h window with IncludeTimeSeries false and a RangeStep
// sized for that span via SetTimeRange.
func DefaultAnalysisOptions() AnalysisOptions {
	opts := AnalysisOptions{
		TimeRange:         24 * time.Hour,
		RangeStep:         1 * time.Minute,
		IncludeTimeSeries: false,
	}
	opts.SetTimeRange(opts.TimeRange)
	return opts
}

// SetTimeRange stores timeRange and sets RangeStep so each range query stays near a safe point
// count for Prometheus (targets roughly 11k samples or fewer per series).
func (opts *AnalysisOptions) SetTimeRange(timeRange time.Duration) {
	opts.TimeRange = timeRange
	const maxPoints = 11000

	totalMinutes := timeRange.Minutes()
	minStepMinutes := totalMinutes / maxPoints

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
		opts.RangeStep = 6 * time.Hour
	}
}

// Validate returns an error if TimeRange or RangeStep are not positive.
func (opts AnalysisOptions) Validate() error {
	if opts.TimeRange <= 0 {
		return fmt.Errorf("TimeRange must be positive, got: %v", opts.TimeRange)
	}
	if opts.RangeStep <= 0 {
		return fmt.Errorf("RangeStep must be positive, got: %v", opts.RangeStep)
	}
	return nil
}

// ContainerAnalysis is utilization, provisioning, and stability for one container, with optional chart series.
type ContainerAnalysis struct {
	ContainerName string             `json:"container_name"`
	Utilization   UtilizationResult  `json:"utilization"`
	Provisioning  ProvisioningResult `json:"provisioning"`
	Stability     StabilityResult    `json:"stability"`
	TimeSeries    *TimeSeries        `json:"time_series,omitempty"`
}

// PodAnalysis is aggregate pod-level compute signals plus per-container analyses and optional pod chart series.
type PodAnalysis struct {
	PodUID      string              `json:"pod_uid"`
	PodName     string              `json:"pod_name"`
	Containers  []ContainerAnalysis `json:"containers"`
	Utilization UtilizationResult   `json:"utilization"`
	Stability   StabilityResult     `json:"stability"`
	TimeSeries  *TimeSeries         `json:"time_series,omitempty"`
}

// WorkloadAnalysis is workload-level utilization and stability, optional per-pod analyses, and optional chart series.
type WorkloadAnalysis struct {
	WorkloadType string            `json:"workload_type"`
	WorkloadName string            `json:"workload_name"`
	Namespace    string            `json:"namespace"`
	Pods         []PodAnalysis     `json:"pods"`
	Utilization  UtilizationResult `json:"utilization"`
	Stability    StabilityResult   `json:"stability"`
	TimeSeries   *TimeSeries       `json:"time_series,omitempty"`
}

// NamespaceAnalysis is namespace-level utilization and stability, workload summaries, and optional chart series.
type NamespaceAnalysis struct {
	Namespace   string             `json:"namespace"`
	Utilization UtilizationResult  `json:"utilization"`
	Stability   StabilityResult    `json:"stability"`
	Workloads   []WorkloadAnalysis `json:"workloads"`
	TimeSeries  *TimeSeries        `json:"time_series,omitempty"`
}

// AnalyzeContainer fetches metrics for container, derives utilization and stability, parses Kubernetes
// resource fields from the DB row, and runs provisioning analysis.
func AnalyzeContainer(ctx context.Context, container *database.Container, opts AnalysisOptions) (ContainerAnalysis, error) {
	if err := opts.Validate(); err != nil {
		return ContainerAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	if container == nil {
		return ContainerAnalysis{}, fmt.Errorf("container cannot be nil")
	}

	metrics, err := fetchContainerMetrics(ctx, container, opts)
	if err != nil {
		return ContainerAnalysis{}, fmt.Errorf("failed to fetch container metrics: %w", err)
	}

	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return ContainerAnalysis{}, apperrors.NewNoMetrics(fmt.Sprintf("container %s", container.Name))
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return ContainerAnalysis{}, fmt.Errorf("failed to analyze utilization: %w", err)
	}

	stability := AnalyzeStability(metrics)

	specs, err := ParseContainerSpecs(container)
	if err != nil {
		return ContainerAnalysis{}, fmt.Errorf("failed to parse container specs: %w", err)
	}

	provisioning := AnalyzeProvisioning(specs, utilization, stability)

	result := ContainerAnalysis{
		ContainerName: container.Name,
		Utilization:   utilization,
		Stability:     stability,
		Provisioning:  provisioning,
	}

	if opts.IncludeTimeSeries {
		result.TimeSeries = &TimeSeries{
			CPU:    metrics.CPU,
			Memory: metrics.Memory,
		}
	}

	return result, nil
}

// AnalyzePod runs AnalyzeContainer for each container in parallel, then attaches pod-level utilization,
// stability, and optional chart series from pod-scoped Prometheus queries.
func AnalyzePod(ctx context.Context, pod *database.Pod, containers []*database.Container, opts AnalysisOptions) (PodAnalysis, error) {
	if err := opts.Validate(); err != nil {
		return PodAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

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

	metrics, err := fetchPodMetrics(ctx, pod, opts)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to fetch pod metrics: %w", err)
	}

	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return PodAnalysis{}, apperrors.NewNoMetrics(fmt.Sprintf("pod %s", pod.Name))
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to analyze pod utilization: %w", err)
	}

	stability := AnalyzeStability(metrics)

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

	if opts.IncludeTimeSeries {
		result.TimeSeries = &TimeSeries{
			CPU:    metrics.CPU,
			Memory: metrics.Memory,
		}
	}

	return result, nil
}

// AnalyzeWorkload aggregates metrics across pods, derives workload-level utilization and stability, and
// optionally fills Pods with per-pod analyses. When includePods is true, nested pod analyses omit chart
// series (IncludeTimeSeries is forced off) so responses stay smaller than attaching series at every level.
func AnalyzeWorkload(ctx context.Context, workloadType string, workloadName string, namespace string, pods []*database.Pod, opts AnalysisOptions, includePods bool) (WorkloadAnalysis, error) {
	if err := opts.Validate(); err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	var podAnalyses []PodAnalysis
	g, gctx := errgroup.WithContext(ctx)
	if includePods {
		podAnalyses = make([]PodAnalysis, len(pods))
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

	metrics, err := fetchWorkloadMetrics(ctx, pods, opts)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("failed to fetch workload metrics: %w", err)
	}

	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return WorkloadAnalysis{}, apperrors.NewNoMetrics(fmt.Sprintf("workload %s/%s", namespace, workloadName))
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("failed to analyze workload utilization: %w", err)
	}

	stability := AnalyzeStability(metrics)

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

	if opts.IncludeTimeSeries {
		result.TimeSeries = &TimeSeries{
			CPU:    metrics.CPU,
			Memory: metrics.Memory,
		}
	}

	return result, nil
}

// AnalyzeNamespace fetches namespace-level metrics, lists workloads (deployments, statefulsets, daemonsets,
// standalone pods, and node-owned “system” pods), and analyzes each workload without nested per-pod trees.
// Workload summaries never include nested chart series, only the top-level namespace result may include
// series when opts.IncludeTimeSeries is true.
func AnalyzeNamespace(ctx context.Context, namespace string, opts AnalysisOptions) (NamespaceAnalysis, error) {
	if err := opts.Validate(); err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	workloadOpts := opts
	workloadOpts.IncludeTimeSeries = false

	var workloadAnalyses []WorkloadAnalysis
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		analyses, err := analyzer.AnalyzeWorkloads(gctx, namespace,
			func(ctx context.Context, kind, name, namespace string, pods []*database.Pod) (WorkloadAnalysis, error) {
				return AnalyzeWorkload(ctx, kind, name, namespace, pods, workloadOpts, false)
			})
		if err != nil {
			return fmt.Errorf("failed to analyze namespace workloads: %w", err)
		}
		workloadAnalyses = analyses
		return nil
	})

	metrics, err := fetchNamespaceMetrics(ctx, namespace, opts)
	if err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("failed to fetch namespace metrics: %w", err)
	}

	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return NamespaceAnalysis{}, apperrors.NewNoMetrics(fmt.Sprintf("namespace %s", namespace))
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("failed to analyze namespace utilization: %w", err)
	}

	stability := AnalyzeStability(metrics)

	if err := g.Wait(); err != nil {
		return NamespaceAnalysis{}, err
	}

	result := NamespaceAnalysis{
		Namespace:   namespace,
		Utilization: utilization,
		Stability:   stability,
		Workloads:   workloadAnalyses,
	}

	if opts.IncludeTimeSeries {
		result.TimeSeries = &TimeSeries{
			CPU:    metrics.CPU,
			Memory: metrics.Memory,
		}
	}

	return result, nil
}

// ParseContainerSpecs converts Kubernetes quantity strings from the database Container model into
// ResourceSpecs (CPU in cores, memory in bytes).
func ParseContainerSpecs(container *database.Container) (ResourceSpecs, error) {
	specs := ResourceSpecs{}

	if container.CPURequest != nil && *container.CPURequest != "" {
		qty, err := resource.ParseQuantity(*container.CPURequest)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse CPU request: %w", err)
		}
		specs.CPURequest = new(qty.AsFloat64Slow())
	}

	if container.CPULimit != nil && *container.CPULimit != "" {
		qty, err := resource.ParseQuantity(*container.CPULimit)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse CPU limit: %w", err)
		}
		specs.CPULimit = new(qty.AsFloat64Slow())
	}

	if container.MemoryRequest != nil && *container.MemoryRequest != "" {
		qty, err := resource.ParseQuantity(*container.MemoryRequest)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse memory request: %w", err)
		}
		specs.MemoryRequest = new(float64(qty.Value()))
	}

	if container.MemoryLimit != nil && *container.MemoryLimit != "" {
		qty, err := resource.ParseQuantity(*container.MemoryLimit)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse memory limit: %w", err)
		}
		specs.MemoryLimit = new(float64(qty.Value()))
	}

	return specs, nil
}
