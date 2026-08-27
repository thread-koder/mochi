package compute

import (
	"context"
	"fmt"
	"time"

	"github.com/thread_koder/mochi/core/internal/analyzer"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/timeseries"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/resource"
)

type AnalysisOptions struct {
	TimeRange         time.Duration // How far back to analyze (default: 24h).
	RangeStep         time.Duration // Step size for Prometheus range queries (default: 1m).
	IncludeTimeSeries bool          // Whether to include raw CPU and memory series for charts (default: false).
}

func DefaultAnalysisOptions() AnalysisOptions {
	opts := AnalysisOptions{
		TimeRange:         24 * time.Hour,
		RangeStep:         1 * time.Minute,
		IncludeTimeSeries: false,
	}
	opts.SetTimeRange(opts.TimeRange)
	return opts
}

// SetTimeRange stores timeRange and sets RangeStep from tiered resolution rules.
func (opts *AnalysisOptions) SetTimeRange(timeRange time.Duration) {
	opts.TimeRange = timeRange
	opts.RangeStep = timeseries.RangeStepForTimeRange(timeRange)
}

// stabilitySubqueryStep returns the step for throttling/PSI subqueries, floored at 5m so
// period averages stay stable without oversampling relative to the rate window.
func (opts AnalysisOptions) stabilitySubqueryStep() time.Duration {
	return max(opts.RangeStep, 5*time.Minute)
}

func (opts AnalysisOptions) MinSamplesForConfidence() int {
	expected := int(opts.TimeRange / opts.RangeStep)
	return max(30, expected/4)
}

func (opts AnalysisOptions) Validate() error {
	if opts.TimeRange <= 0 {
		return fmt.Errorf("TimeRange must be positive, got: %v", opts.TimeRange)
	}
	if opts.RangeStep <= 0 {
		return fmt.Errorf("RangeStep must be positive, got: %v", opts.RangeStep)
	}
	return nil
}

type ContainerAnalysis struct {
	ContainerName string             `json:"container_name"`
	Utilization   UtilizationResult  `json:"utilization"`
	Provisioning  ProvisioningResult `json:"provisioning"`
	Stability     StabilityResult    `json:"stability"`
	TimeSeries    *TimeSeries        `json:"time_series,omitempty"`
}

type PodAnalysis struct {
	PodUID      string              `json:"pod_uid"`
	PodName     string              `json:"pod_name"`
	Containers  []ContainerAnalysis `json:"containers"`
	Utilization UtilizationResult   `json:"utilization"`
	Stability   StabilityResult     `json:"stability"`
	TimeSeries  *TimeSeries         `json:"time_series,omitempty"`
}

type WorkloadAnalysis struct {
	WorkloadType string            `json:"workload_type"`
	WorkloadName string            `json:"workload_name"`
	Namespace    string            `json:"namespace"`
	Pods         []PodAnalysis     `json:"pods"`
	Utilization  UtilizationResult `json:"utilization"`
	Stability    StabilityResult   `json:"stability"`
	TimeSeries   *TimeSeries       `json:"time_series,omitempty"`
}

type NamespaceAnalysis struct {
	Namespace   string             `json:"namespace"`
	Utilization UtilizationResult  `json:"utilization"`
	Stability   StabilityResult    `json:"stability"`
	Workloads   []WorkloadAnalysis `json:"workloads"`
	TimeSeries  *TimeSeries        `json:"time_series,omitempty"`
}

func AnalyzeContainer(ctx context.Context, container *database.Container, opts AnalysisOptions) (ContainerAnalysis, error) {
	if err := opts.Validate(); err != nil {
		return ContainerAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	if container == nil {
		return ContainerAnalysis{}, fmt.Errorf("container cannot be nil")
	}

	metrics, err := fetchContainerMetrics(ctx, container, opts)
	if err != nil {
		return ContainerAnalysis{}, fmt.Errorf("failed to analyze container %s: %w", container.Name, err)
	}

	if !hasAnalyzableComputeMetrics(metrics) {
		return ContainerAnalysis{}, apperrors.NewNoMetrics(fmt.Sprintf("container %s", container.Name))
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return ContainerAnalysis{}, fmt.Errorf("failed to analyze container %s: %w", container.Name, err)
	}

	stability := AnalyzeStability(metrics)

	specs, err := ParseContainerSpecs(container)
	if err != nil {
		return ContainerAnalysis{}, fmt.Errorf("failed to analyze container %s: %w", container.Name, err)
	}

	provisioning := AnalyzeProvisioning(specs, utilization, stability, opts.MinSamplesForConfidence())

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

func AnalyzePod(ctx context.Context, pod *database.Pod, containers []*database.Container, opts AnalysisOptions) (PodAnalysis, error) {
	if err := opts.Validate(); err != nil {
		return PodAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	if pod == nil {
		return PodAnalysis{}, fmt.Errorf("pod cannot be nil")
	}

	g, gctx := errgroup.WithContext(ctx)
	var containerAnalyses []ContainerAnalysis
	g.Go(func() error {
		analyses, err := analyzer.SkipNoMetrics(gctx, containers, func(ctx context.Context, container *database.Container) (ContainerAnalysis, error) {
			return AnalyzeContainer(ctx, container, opts)
		})
		if err != nil {
			return err
		}
		containerAnalyses = analyses
		return nil
	})

	metrics, err := fetchPodMetrics(ctx, pod, opts)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to analyze pod %s: %w", pod.Name, err)
	}

	if !hasAnalyzableComputeMetrics(metrics) {
		return PodAnalysis{}, apperrors.NewNoMetrics(fmt.Sprintf("pod %s", pod.Name))
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to analyze pod %s: %w", pod.Name, err)
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

func AnalyzeWorkload(ctx context.Context, workloadType string, workloadName string, namespace string, pods database.PodsForAnalysis, opts AnalysisOptions, includePods bool) (WorkloadAnalysis, error) {
	if err := opts.Validate(); err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	var podAnalyses []PodAnalysis
	g, gctx := errgroup.WithContext(ctx)
	if includePods {
		podOpts := opts
		podOpts.IncludeTimeSeries = false
		g.Go(func() error {
			analyses, err := analyzer.SkipNoMetrics(gctx, pods.Live, func(ctx context.Context, pod *database.Pod) (PodAnalysis, error) {
				containers, err := database.GetContainersByPodUID(ctx, pod.UID)
				if err != nil {
					return PodAnalysis{}, err
				}
				return AnalyzePod(ctx, pod, containers, podOpts)
			})
			if err != nil {
				return err
			}
			podAnalyses = analyses
			return nil
		})
	}

	metrics, err := fetchWorkloadMetrics(ctx, pods.All, opts)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("failed to analyze workload %s/%s/%s: %w", workloadType, namespace, workloadName, err)
	}

	if !hasAnalyzableComputeMetrics(metrics) {
		return WorkloadAnalysis{}, apperrors.NewNoMetrics(fmt.Sprintf("workload %s/%s", namespace, workloadName))
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("failed to analyze workload %s/%s/%s: %w", workloadType, namespace, workloadName, err)
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

func AnalyzeNamespace(ctx context.Context, namespace string, opts AnalysisOptions) (NamespaceAnalysis, error) {
	if err := opts.Validate(); err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	workloadOpts := opts
	workloadOpts.IncludeTimeSeries = false
	since := time.Now().Add(-opts.TimeRange)

	var workloadAnalyses []WorkloadAnalysis
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		analyses, err := analyzer.AnalyzeWorkloads(gctx, namespace, since,
			func(ctx context.Context, kind, name, namespace string, pods database.PodsForAnalysis) (WorkloadAnalysis, error) {
				return AnalyzeWorkload(ctx, kind, name, namespace, pods, workloadOpts, false)
			})
		if err != nil {
			return err
		}
		workloadAnalyses = analyses
		return nil
	})

	metrics, err := fetchNamespaceMetrics(ctx, namespace, opts)
	if err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("failed to analyze namespace %s: %w", namespace, err)
	}

	if !hasAnalyzableComputeMetrics(metrics) {
		return NamespaceAnalysis{}, apperrors.NewNoMetrics(fmt.Sprintf("namespace %s", namespace))
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("failed to analyze namespace %s: %w", namespace, err)
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

	if container.CPURequest != nil {
		qty, err := resource.ParseQuantity(*container.CPURequest)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse CPU request: %w", err)
		}
		specs.CPURequest = new(qty.AsFloat64Slow())
	}

	if container.CPULimit != nil {
		qty, err := resource.ParseQuantity(*container.CPULimit)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse CPU limit: %w", err)
		}
		specs.CPULimit = new(qty.AsFloat64Slow())
	}

	if container.MemoryRequest != nil {
		qty, err := resource.ParseQuantity(*container.MemoryRequest)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse memory request: %w", err)
		}
		specs.MemoryRequest = new(float64(qty.Value()))
	}

	if container.MemoryLimit != nil {
		qty, err := resource.ParseQuantity(*container.MemoryLimit)
		if err != nil {
			return ResourceSpecs{}, fmt.Errorf("failed to parse memory limit: %w", err)
		}
		specs.MemoryLimit = new(float64(qty.Value()))
	}

	return specs, nil
}
