// Package disk loads pod, workload, and namespace disk metrics from Prometheus,
// summarizes utilization, and optionally attach raw byte-rate and operation-rate samples for charting.
package disk

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thread_koder/mochi/internal/database"
	"golang.org/x/sync/errgroup"
)

// AnalysisOptions controls the analysis window, Prometheus range-query resolution,
// and whether results include raw read/write series for charts.
type AnalysisOptions struct {
	TimeRange         time.Duration // How far back to analyze (default: 24h).
	RangeStep         time.Duration // Step size for Prometheus range queries (default: 1m).
	IncludeTimeSeries bool          // Whether to include raw read/write series for charts (default: false).
}

// DefaultAnalysisOptions returns a 24h window with IncludeTimeSeries false and a
// RangeStep sized for that span via SetTimeRange.
func DefaultAnalysisOptions() AnalysisOptions {
	opts := AnalysisOptions{
		TimeRange:         24 * time.Hour,
		RangeStep:         1 * time.Minute,
		IncludeTimeSeries: false,
	}
	opts.SetTimeRange(opts.TimeRange)
	return opts
}

// SetTimeRange stores timeRange and sets RangeStep so each range query stays near
// a safe point count for Prometheus (targets roughly 11k samples or fewer per series).
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

// PodAnalysis is disk utilization for one pod plus optional chart series.
type PodAnalysis struct {
	PodUID      string            `json:"pod_uid"`
	PodName     string            `json:"pod_name"`
	Utilization UtilizationResult `json:"utilization"`
	TimeSeries  *TimeSeries       `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

// WorkloadAnalysis is aggregated disk utilization for a workload plus optional
// per-pod analyses and optional chart series.
type WorkloadAnalysis struct {
	WorkloadType string            `json:"workload_type"`
	WorkloadName string            `json:"workload_name"`
	Namespace    string            `json:"namespace"`
	Pods         []PodAnalysis     `json:"pods"`
	Utilization  UtilizationResult `json:"utilization"`
	TimeSeries   *TimeSeries       `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

// NamespaceAnalysis is namespace-level utilization plus workload analyses and
// optional namespace chart series.
type NamespaceAnalysis struct {
	Namespace   string             `json:"namespace"`
	Utilization UtilizationResult  `json:"utilization"`
	Workloads   []WorkloadAnalysis `json:"workloads"`
	TimeSeries  *TimeSeries        `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

// AnalyzePod fetches disk metrics for pod, summarizes utilization, and attaches
// optional pod chart series.
func AnalyzePod(ctx context.Context, pod *database.Pod, opts AnalysisOptions) (PodAnalysis, error) {
	if err := opts.Validate(); err != nil {
		return PodAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	if pod == nil {
		return PodAnalysis{}, fmt.Errorf("pod cannot be nil")
	}

	metrics, err := fetchPodMetrics(ctx, pod, opts)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to fetch pod metrics: %w", err)
	}

	if len(metrics.ReadBytes) == 0 && len(metrics.WriteBytes) == 0 {
		return PodAnalysis{}, fmt.Errorf("no metrics available for pod %s", pod.Name)
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to analyze utilization: %w", err)
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

// AnalyzeWorkload fetches merged metrics across pods, summarizes workload-level utilization,
// attaches optional workload chart series, and optionally fills Pods with per-pod analyses.
// When includePods is true, each pod analysis omits TimeSeries
// so the JSON does not embed full chart series under every workload and pod.
func AnalyzeWorkload(ctx context.Context, workloadType string, workloadName string, namespace string, pods []*database.Pod, opts AnalysisOptions, includePods bool) (WorkloadAnalysis, error) {
	if err := opts.Validate(); err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	if len(pods) == 0 {
		return WorkloadAnalysis{}, fmt.Errorf("no pods found for workload %s/%s", namespace, workloadName)
	}

	var podAnalyses []PodAnalysis
	g, gctx := errgroup.WithContext(ctx)
	if includePods {
		podAnalyses = make([]PodAnalysis, len(pods))
		podOpts := opts
		// Heavy chart payloads: keep series only at the workload level when includePods is on.
		podOpts.IncludeTimeSeries = false
		for i, pod := range pods {
			g.Go(func() error {
				podAnalysis, err := AnalyzePod(gctx, pod, podOpts)
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

	if len(metrics.ReadBytes) == 0 && len(metrics.WriteBytes) == 0 {
		return WorkloadAnalysis{}, fmt.Errorf("no metrics available for workload %s/%s", namespace, workloadName)
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("failed to analyze workload utilization: %w", err)
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

// AnalyzeNamespace returns namespace-wide utilization, a workload-level
// breakdown, and attaches optional namespace chart series, Child WorkloadAnalysis
// never include TimeSeries so responses stay smaller.
func AnalyzeNamespace(ctx context.Context, namespace string, opts AnalysisOptions) (NamespaceAnalysis, error) {
	if err := opts.Validate(); err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	workloadOpts := opts
	workloadOpts.IncludeTimeSeries = false

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

	metrics, err := fetchNamespaceMetrics(ctx, namespace, opts)
	if err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("failed to fetch namespace metrics: %w", err)
	}

	if len(metrics.ReadBytes) == 0 && len(metrics.WriteBytes) == 0 {
		return NamespaceAnalysis{}, fmt.Errorf("no metrics available for namespace %s", namespace)
	}

	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("failed to analyze namespace utilization: %w", err)
	}

	if err := g.Wait(); err != nil {
		return NamespaceAnalysis{}, err
	}

	result := NamespaceAnalysis{
		Namespace:   namespace,
		Utilization: utilization,
		Workloads:   workloads,
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

// analyzeNamespaceWorkloads walks Deployments, StatefulSets, DaemonSets,
// standalone Pods, and Pods owned by Nodes (system pods like kube-proxy and
// coredns), and returns one WorkloadAnalysis per entry at workload granularity
// only.
func analyzeNamespaceWorkloads(ctx context.Context, namespace string, opts AnalysisOptions) ([]WorkloadAnalysis, error) {
	workloads := make([]WorkloadAnalysis, 0)
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(ctx)

	analyzeSingle := func(kind, name string, pods []*database.Pod) {
		g.Go(func() error {
			if len(pods) == 0 {
				return nil
			}
			analysis, err := AnalyzeWorkload(gctx, kind, name, namespace, pods, opts, false)
			if err != nil {
				// Some workloads never expose disk metrics,
				// so skip those to return all other workloads that do have data.
				if strings.Contains(err.Error(), "no metrics available") {
					return nil
				}
				return fmt.Errorf("failed to analyze workload %s/%s: %w", kind, name, err)
			}

			mu.Lock()
			workloads = append(workloads, analysis)
			mu.Unlock()
			return nil
		})
	}

	// Nested g.Go calls share this errgroup, and Wait still collects every task
	// queued before it returns.
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

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return workloads, nil
}
