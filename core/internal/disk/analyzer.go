package disk

import (
	"context"
	"fmt"
	"time"

	"github.com/thread_koder/mochi/core/internal/analyzer"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/timeseries"
	"golang.org/x/sync/errgroup"
)

type AnalysisOptions struct {
	TimeRange         time.Duration // How far back to analyze (default: 24h).
	RangeStep         time.Duration // Step size for Prometheus range queries (default: 1m).
	IncludeTimeSeries bool          // Whether to include raw read/write series for charts (default: false).
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

func (opts AnalysisOptions) Validate() error {
	if opts.TimeRange <= 0 {
		return fmt.Errorf("TimeRange must be positive, got: %v", opts.TimeRange)
	}
	if opts.RangeStep <= 0 {
		return fmt.Errorf("RangeStep must be positive, got: %v", opts.RangeStep)
	}
	return nil
}

type PodAnalysis struct {
	PodUID      string            `json:"pod_uid"`
	PodName     string            `json:"pod_name"`
	Utilization UtilizationResult `json:"utilization"`
	TimeSeries  *TimeSeries       `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

type WorkloadAnalysis struct {
	WorkloadType string            `json:"workload_type"`
	WorkloadName string            `json:"workload_name"`
	Namespace    string            `json:"namespace"`
	Pods         []PodAnalysis     `json:"pods"`
	Utilization  UtilizationResult `json:"utilization"`
	TimeSeries   *TimeSeries       `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

type NamespaceAnalysis struct {
	Namespace   string             `json:"namespace"`
	Utilization UtilizationResult  `json:"utilization"`
	Workloads   []WorkloadAnalysis `json:"workloads"`
	TimeSeries  *TimeSeries        `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

func AnalyzePod(ctx context.Context, pod *database.Pod, opts AnalysisOptions) (PodAnalysis, error) {
	if err := opts.Validate(); err != nil {
		return PodAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	if pod == nil {
		return PodAnalysis{}, fmt.Errorf("pod cannot be nil")
	}

	metrics, err := fetchPodMetrics(ctx, pod, opts)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to analyze pod %s: %w", pod.Name, err)
	}

	if !hasAnalyzableDiskMetrics(metrics) {
		return PodAnalysis{}, apperrors.NewNoMetrics(fmt.Sprintf("pod %s", pod.Name))
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to analyze pod %s: %w", pod.Name, err)
	}

	result := PodAnalysis{
		PodUID:      pod.UID,
		PodName:     pod.Name,
		Utilization: utilization,
	}

	if opts.IncludeTimeSeries {
		result.TimeSeries = &TimeSeries{
			ReadBytes:  metrics.ReadBytes,
			WriteBytes: metrics.WriteBytes,
			ReadOps:    metrics.ReadOps,
			WriteOps:   metrics.WriteOps,
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
				return AnalyzePod(ctx, pod, podOpts)
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

	if !hasAnalyzableDiskMetrics(metrics) {
		return WorkloadAnalysis{}, apperrors.NewNoMetrics(fmt.Sprintf("workload %s/%s", namespace, workloadName))
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("failed to analyze workload %s/%s/%s: %w", workloadType, namespace, workloadName, err)
	}

	if err := g.Wait(); err != nil {
		return WorkloadAnalysis{}, err
	}

	result := WorkloadAnalysis{
		WorkloadType: workloadType,
		WorkloadName: workloadName,
		Namespace:    namespace,
		Pods:         podAnalyses,
		Utilization:  utilization,
	}

	if opts.IncludeTimeSeries {
		result.TimeSeries = &TimeSeries{
			ReadBytes:  metrics.ReadBytes,
			WriteBytes: metrics.WriteBytes,
			ReadOps:    metrics.ReadOps,
			WriteOps:   metrics.WriteOps,
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

	if !hasAnalyzableDiskMetrics(metrics) {
		return NamespaceAnalysis{}, apperrors.NewNoMetrics(fmt.Sprintf("namespace %s", namespace))
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("failed to analyze namespace %s: %w", namespace, err)
	}

	if err := g.Wait(); err != nil {
		return NamespaceAnalysis{}, err
	}

	result := NamespaceAnalysis{
		Namespace:   namespace,
		Utilization: utilization,
		Workloads:   workloadAnalyses,
	}

	if opts.IncludeTimeSeries {
		result.TimeSeries = &TimeSeries{
			ReadBytes:  metrics.ReadBytes,
			WriteBytes: metrics.WriteBytes,
			ReadOps:    metrics.ReadOps,
			WriteOps:   metrics.WriteOps,
		}
	}

	return result, nil
}
