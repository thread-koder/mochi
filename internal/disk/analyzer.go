package disk

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/thread_koder/mochi/internal/database"
	"github.com/thread_koder/mochi/internal/prometheus"
	"github.com/thread_koder/mochi/internal/timeseries"
	"golang.org/x/sync/errgroup"
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

// Represents analysis results for a pod
type PodAnalysis struct {
	PodUID      string            `json:"pod_uid"`
	PodName     string            `json:"pod_name"`
	Utilization UtilizationResult `json:"utilization"`
	TimeSeries  *TimeSeries       `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

// Represents analysis results for a workload
type WorkloadAnalysis struct {
	WorkloadType string            `json:"workload_type"`
	WorkloadName string            `json:"workload_name"`
	Namespace    string            `json:"namespace"`
	Pods         []PodAnalysis     `json:"pods"`
	Utilization  UtilizationResult `json:"utilization"`
	TimeSeries   *TimeSeries       `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

// Represents analysis results for a namespace
type NamespaceAnalysis struct {
	Namespace   string             `json:"namespace"`
	Utilization UtilizationResult  `json:"utilization"`
	Workloads   []WorkloadAnalysis `json:"workloads"`
	TimeSeries  *TimeSeries        `json:"time_series,omitempty"` // Optional: raw datapoints for charting
}

// Analyzes a pod's disk utilization
func AnalyzePod(ctx context.Context, pod *database.Pod, opts AnalysisOptions) (PodAnalysis, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return PodAnalysis{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	if pod == nil {
		return PodAnalysis{}, fmt.Errorf("pod cannot be nil")
	}

	// Fetch metrics from Prometheus
	metrics, err := fetchPodMetrics(ctx, pod, opts)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to fetch pod metrics: %w", err)
	}

	// Validate we have some metrics
	if len(metrics.ReadBytes) == 0 && len(metrics.WriteBytes) == 0 {
		return PodAnalysis{}, fmt.Errorf("no metrics available for pod %s", pod.Name)
	}

	// Analyze utilization
	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return PodAnalysis{}, fmt.Errorf("failed to analyze utilization: %w", err)
	}

	result := PodAnalysis{
		PodUID:      pod.UID,
		PodName:     pod.Name,
		Utilization: utilization,
	}

	// Include time series if requested
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
				podAnalysis, err := AnalyzePod(gctx, pod, podOpts)
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
	if len(metrics.ReadBytes) == 0 && len(metrics.WriteBytes) == 0 {
		return WorkloadAnalysis{}, fmt.Errorf("no metrics available for workload %s/%s", namespace, workloadName)
	}

	// Analyze utilization
	utilization, err := AnalyzeUtilization(metrics)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("failed to analyze workload utilization: %w", err)
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
	}

	// Include time series if requested
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
	if len(metrics.ReadBytes) == 0 && len(metrics.WriteBytes) == 0 {
		return NamespaceAnalysis{}, fmt.Errorf("no metrics available for namespace %s", namespace)
	}

	// Analyze utilization
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
		Workloads:   workloads,
	}

	// Include time series if requested
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
				// Skip workloads with no metrics
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

// Fetches pod disk metrics
func fetchPodMetrics(ctx context.Context, pod *database.Pod, opts AnalysisOptions) (DiskMetrics, error) {
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

	var (
		readBytesMatrix  model.Matrix
		writeBytesMatrix model.Matrix
		readOpsMatrix    model.Matrix
		writeOpsMatrix   model.Matrix
	)

	// Execute all queries in parallel
	g, gctx := errgroup.WithContext(ctx)

	// Query read bytes
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodDiskReadBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read bytes metrics: %w", err)
		}
		readBytesMatrix = matrix
		return nil
	})

	// Query write bytes
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodDiskWriteBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write bytes metrics: %w", err)
		}
		writeBytesMatrix = matrix
		return nil
	})

	// Query read ops
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodDiskReadOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read ops metrics: %w", err)
		}
		readOpsMatrix = matrix
		return nil
	})

	// Query write ops
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodDiskWriteOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write ops metrics: %w", err)
		}
		writeOpsMatrix = matrix
		return nil
	})

	// Wait for all queries to be completed and check for errors
	if err := g.Wait(); err != nil {
		return DiskMetrics{}, err
	}

	return DiskMetrics{
		ReadBytes:  timeseries.MatrixToDataPoints(readBytesMatrix),
		WriteBytes: timeseries.MatrixToDataPoints(writeBytesMatrix),
		ReadOps:    timeseries.MatrixToDataPoints(readOpsMatrix),
		WriteOps:   timeseries.MatrixToDataPoints(writeOpsMatrix),
	}, nil
}

// Aggregates metrics from all pods in a workload
func fetchWorkloadMetrics(ctx context.Context, pods []*database.Pod, opts AnalysisOptions) (DiskMetrics, error) {
	if len(pods) == 0 {
		return DiskMetrics{}, fmt.Errorf("no pods found for workload")
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
		ReadBytes  []timeseries.DataPoint
		WriteBytes []timeseries.DataPoint
		ReadOps    []timeseries.DataPoint
		WriteOps   []timeseries.DataPoint
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
			var (
				readBytesMatrix  model.Matrix
				writeBytesMatrix model.Matrix
				readOpsMatrix    model.Matrix
				writeOpsMatrix   model.Matrix
			)

			// Create a new error group for this pod
			podG, podCtx := errgroup.WithContext(gctx)

			// Query read bytes
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskReadBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query read bytes metrics: %w", err)
				}
				readBytesMatrix = matrix
				return nil
			})

			// Query write bytes
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskWriteBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query write bytes metrics: %w", err)
				}
				writeBytesMatrix = matrix
				return nil
			})

			// Query read ops
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskReadOpsRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query read ops metrics: %w", err)
				}
				readOpsMatrix = matrix
				return nil
			})

			// Query write ops
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskWriteOpsRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query write ops metrics: %w", err)
				}
				writeOpsMatrix = matrix
				return nil
			})

			// Wait for all queries to be completed and check for errors
			if err := podG.Wait(); err != nil {
				return err
			}

			results[i] = podMetrics{
				ReadBytes:  timeseries.MatrixToDataPoints(readBytesMatrix),
				WriteBytes: timeseries.MatrixToDataPoints(writeBytesMatrix),
				ReadOps:    timeseries.MatrixToDataPoints(readOpsMatrix),
				WriteOps:   timeseries.MatrixToDataPoints(writeOpsMatrix),
			}
			return nil
		})
	}

	// Wait for all queries to be completed and check for errors
	if err := g.Wait(); err != nil {
		return DiskMetrics{}, err
	}

	// Aggregate metrics across pods
	var readBytes, writeBytes []timeseries.DataPoint
	var readOps, writeOps []timeseries.DataPoint
	for _, p := range results {
		readBytes = timeseries.MergeDataPointsByTime(readBytes, p.ReadBytes)
		writeBytes = timeseries.MergeDataPointsByTime(writeBytes, p.WriteBytes)
		readOps = timeseries.MergeDataPointsByTime(readOps, p.ReadOps)
		writeOps = timeseries.MergeDataPointsByTime(writeOps, p.WriteOps)
	}

	return DiskMetrics{
		ReadBytes:  readBytes,
		WriteBytes: writeBytes,
		ReadOps:    readOps,
		WriteOps:   writeOps,
	}, nil
}

// Fetches namespace disk metrics
func fetchNamespaceMetrics(ctx context.Context, namespace string, opts AnalysisOptions) (DiskMetrics, error) {
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

	var (
		readBytesMatrix  model.Matrix
		writeBytesMatrix model.Matrix
		readOpsMatrix    model.Matrix
		writeOpsMatrix   model.Matrix
	)

	// Execute queries in parallel
	g, gctx := errgroup.WithContext(ctx)

	// Query namespace read bytes
	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceDiskReadBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read bytes metrics: %w", err)
		}
		readBytesMatrix = matrix
		return nil
	})

	// Query namespace write bytes
	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceDiskWriteBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write bytes metrics: %w", err)
		}
		writeBytesMatrix = matrix
		return nil
	})

	// Query namespace read ops
	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceDiskReadOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read ops metrics: %w", err)
		}
		readOpsMatrix = matrix
		return nil
	})

	// Query namespace write ops
	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceDiskWriteOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write ops metrics: %w", err)
		}
		writeOpsMatrix = matrix
		return nil
	})

	// Wait for all queries to be completed and check for errors
	if err := g.Wait(); err != nil {
		return DiskMetrics{}, err
	}

	return DiskMetrics{
		ReadBytes:  timeseries.MatrixToDataPoints(readBytesMatrix),
		WriteBytes: timeseries.MatrixToDataPoints(writeBytesMatrix),
		ReadOps:    timeseries.MatrixToDataPoints(readOpsMatrix),
		WriteOps:   timeseries.MatrixToDataPoints(writeOpsMatrix),
	}, nil
}
