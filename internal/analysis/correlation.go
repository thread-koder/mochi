package analysis

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/thread_koder/mochi/internal/database"
	"github.com/thread_koder/mochi/internal/prometheus"
	"github.com/thread_koder/mochi/internal/timeseries"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Holds configuration for correlation analysis
type CorrelationOptions struct {
	TimeRange time.Duration // How far back to analyze (default: 24h)
	RangeStep time.Duration // Step size for range queries (default: 1m)
	MaxLag    time.Duration // Maximum lag to test for cross-correlation (default: 5m)
	LagStep   time.Duration // Step size for lag testing (default: 1m)
}

// Returns default correlation options
func DefaultCorrelationOptions() CorrelationOptions {
	opts := CorrelationOptions{
		TimeRange: 24 * time.Hour,
		RangeStep: 1 * time.Minute,
		MaxLag:    5 * time.Minute,
		LagStep:   1 * time.Minute,
	}
	opts.SetTimeRange(opts.TimeRange)
	return opts
}

// Sets the time range and adjusts the step size to respect Prometheus limits
func (opts *CorrelationOptions) SetTimeRange(timeRange time.Duration) {
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
		opts.RangeStep = 6 * time.Hour
	}
}

// Validates correlation options
func (opts CorrelationOptions) Validate() error {
	if opts.TimeRange <= 0 {
		return fmt.Errorf("TimeRange must be positive, got: %v", opts.TimeRange)
	}
	if opts.RangeStep <= 0 {
		return fmt.Errorf("RangeStep must be positive, got: %v", opts.RangeStep)
	}
	if opts.MaxLag < 0 {
		return fmt.Errorf("MaxLag cannot be negative, got: %v", opts.MaxLag)
	}
	if opts.LagStep <= 0 {
		return fmt.Errorf("LagStep must be positive, got: %v", opts.LagStep)
	}
	return nil
}

// Represents a pair of metrics being correlated
type MetricPair struct {
	MetricA string `json:"metric_a"`
	MetricB string `json:"metric_b"`
}

// Represents the correlation between a pair of metrics
type PairCorrelation struct {
	Pair           MetricPair                        `json:"pair"`
	Correlation    timeseries.CrossCorrelationResult `json:"correlation"`
	Interpretation string                            `json:"interpretation"`
	DataAvailable  bool                              `json:"data_available"`
}

// Represents workload characterization based on correlation patterns
type WorkloadType string

const (
	WorkloadTypeComputeBound   WorkloadType = "compute-bound"
	WorkloadTypeRequestDriven  WorkloadType = "request-driven"
	WorkloadTypeDataProcessing WorkloadType = "data-processing"
	WorkloadTypePassThrough    WorkloadType = "pass-through"
	WorkloadTypeCacheHeavy     WorkloadType = "cache-heavy"
	WorkloadTypeMixed          WorkloadType = "mixed"
)

// Represents the result of workload correlation analysis
type WorkloadCorrelationResult struct {
	WorkloadType      string            `json:"workload_type"`
	WorkloadName      string            `json:"workload_name"`
	Namespace         string            `json:"namespace"`
	Correlations      []PairCorrelation `json:"correlations"`
	CharacterizedAs   WorkloadType      `json:"characterized_as"`
	Insights          []string          `json:"insights"`
	OptimizationHints []string          `json:"optimization_hints"`
	AnalysisPeriod    time.Duration     `json:"analysis_period"`
	DataPointsUsed    int               `json:"data_points_used"`
}

// Holds all metrics needed for correlation analysis
type correlationMetrics struct {
	CPU             []timeseries.DataPoint
	Memory          []timeseries.DataPoint
	NetworkReceive  []timeseries.DataPoint
	NetworkTransmit []timeseries.DataPoint
	DiskRead        []timeseries.DataPoint
	DiskWrite       []timeseries.DataPoint
}

// All metric pairs to analyze (12 pairs)
var metricPairs = []MetricPair{
	{MetricA: "cpu", MetricB: "memory"},
	{MetricA: "cpu", MetricB: "network_receive"},
	{MetricA: "cpu", MetricB: "network_transmit"},
	{MetricA: "cpu", MetricB: "disk_read"},
	{MetricA: "cpu", MetricB: "disk_write"},
	{MetricA: "memory", MetricB: "network_receive"},
	{MetricA: "memory", MetricB: "network_transmit"},
	{MetricA: "memory", MetricB: "disk_read"},
	{MetricA: "memory", MetricB: "disk_write"},
	{MetricA: "network_receive", MetricB: "disk_write"},
	{MetricA: "network_receive", MetricB: "network_transmit"},
	{MetricA: "disk_read", MetricB: "disk_write"},
}

// Analyzes correlations between all metrics for a workload
func AnalyzeWorkloadCorrelations(ctx context.Context, workloadType string, workloadName string, namespace string, pods []*database.Pod, opts CorrelationOptions) (WorkloadCorrelationResult, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return WorkloadCorrelationResult{}, fmt.Errorf("invalid correlation options: %w", err)
	}

	if len(pods) == 0 {
		return WorkloadCorrelationResult{}, fmt.Errorf("no pods found for workload %s/%s", namespace, workloadName)
	}

	// Fetch all metrics
	metrics, err := fetchWorkloadMetrics(ctx, pods, opts)
	if err != nil {
		return WorkloadCorrelationResult{}, fmt.Errorf("failed to fetch workload metrics: %w", err)
	}

	// Calculate correlations for all pairs
	correlations := make([]PairCorrelation, 0, len(metricPairs))

	maxDataPoints := 0
	for _, pair := range metricPairs {
		dataA := getMetricData(metrics, pair.MetricA)
		dataB := getMetricData(metrics, pair.MetricB)

		pairCorr := PairCorrelation{
			Pair:          pair,
			DataAvailable: len(dataA) >= 2 && len(dataB) >= 2,
		}

		if pairCorr.DataAvailable {
			crossCorr, err := timeseries.CalculateCrossCorrelation(dataA, dataB, opts.MaxLag, opts.LagStep)
			if err != nil {
				pairCorr.DataAvailable = false
				pairCorr.Interpretation = fmt.Sprintf("Could not calculate correlation: %v", err)
			} else {
				pairCorr.Correlation = crossCorr
				pairCorr.Interpretation = interpretCorrelation(pair, crossCorr)
				if crossCorr.ZeroLag.SampleSize > maxDataPoints {
					maxDataPoints = crossCorr.ZeroLag.SampleSize
				}
			}
		} else {
			pairCorr.Interpretation = "Insufficient data for correlation analysis"
		}

		correlations = append(correlations, pairCorr)
	}

	// Characterize workload based on correlation patterns
	characterization := characterizeWorkload(correlations)

	// Generate insights
	insights := generateInsights(correlations)

	// Generate optimization hints based on characterization
	optimizationHints := generateOptimizationHints(characterization, correlations)

	return WorkloadCorrelationResult{
		WorkloadType:      workloadType,
		WorkloadName:      workloadName,
		Namespace:         namespace,
		Correlations:      correlations,
		CharacterizedAs:   characterization,
		Insights:          insights,
		OptimizationHints: optimizationHints,
		AnalysisPeriod:    opts.TimeRange,
		DataPointsUsed:    maxDataPoints,
	}, nil
}

// Fetches all metrics needed for correlation analysis
func fetchWorkloadMetrics(ctx context.Context, pods []*database.Pod, opts CorrelationOptions) (correlationMetrics, error) {
	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	var metrics correlationMetrics
	g, gctx := errgroup.WithContext(ctx)

	// Per-pod metrics
	type podMetrics struct {
		CPU             []timeseries.DataPoint
		Memory          []timeseries.DataPoint
		NetworkReceive  []timeseries.DataPoint
		NetworkTransmit []timeseries.DataPoint
		DiskRead        []timeseries.DataPoint
		DiskWrite       []timeseries.DataPoint
	}
	results := make([]podMetrics, len(pods))

	// Fetch metrics for all pods in parallel
	for i, pod := range pods {
		queryOpts := prometheus.QueryOptions{
			Namespace:     pod.Namespace,
			Pod:           pod.Name,
			RangeDuration: "5m",
		}

		g.Go(func() error {
			var (
				cpuMatrix             model.Matrix
				memoryMatrix          model.Matrix
				networkReceiveMatrix  model.Matrix
				networkTransmitMatrix model.Matrix
				diskReadMatrix        model.Matrix
				diskWriteMatrix       model.Matrix
			)

			podG, podCtx := errgroup.WithContext(gctx)

			// Query CPU metrics
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodCPURange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query CPU metrics: %w", err)
				}
				cpuMatrix = matrix
				return nil
			})

			// Query memory metrics
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodMemoryRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query memory metrics: %w", err)
				}
				memoryMatrix = matrix
				return nil
			})

			// Query network receive metrics
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodNetworkReceiveBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return nil
				}
				networkReceiveMatrix = matrix
				return nil
			})

			// Query network transmit metrics
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodNetworkTransmitBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return nil
				}
				networkTransmitMatrix = matrix
				return nil
			})

			// Query disk read metrics
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskReadBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return nil
				}
				diskReadMatrix = matrix
				return nil
			})

			// Query disk write metrics
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskWriteBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return nil
				}
				diskWriteMatrix = matrix
				return nil
			})

			// Wait for all queries to be completed and check for errors
			if err := podG.Wait(); err != nil {
				return err
			}

			results[i] = podMetrics{
				CPU:             timeseries.MatrixToDataPoints(cpuMatrix),
				Memory:          timeseries.MatrixToDataPoints(memoryMatrix),
				NetworkReceive:  timeseries.MatrixToDataPoints(networkReceiveMatrix),
				NetworkTransmit: timeseries.MatrixToDataPoints(networkTransmitMatrix),
				DiskRead:        timeseries.MatrixToDataPoints(diskReadMatrix),
				DiskWrite:       timeseries.MatrixToDataPoints(diskWriteMatrix),
			}
			return nil
		})
	}

	// Wait for all queries to be completed and check for errors
	if err := g.Wait(); err != nil {
		return correlationMetrics{}, err
	}

	// Aggregate metrics across pods
	for _, pm := range results {
		metrics.CPU = timeseries.MergeDataPointsByTime(metrics.CPU, pm.CPU)
		metrics.Memory = timeseries.MergeDataPointsByTime(metrics.Memory, pm.Memory)
		metrics.NetworkReceive = timeseries.MergeDataPointsByTime(metrics.NetworkReceive, pm.NetworkReceive)
		metrics.NetworkTransmit = timeseries.MergeDataPointsByTime(metrics.NetworkTransmit, pm.NetworkTransmit)
		metrics.DiskRead = timeseries.MergeDataPointsByTime(metrics.DiskRead, pm.DiskRead)
		metrics.DiskWrite = timeseries.MergeDataPointsByTime(metrics.DiskWrite, pm.DiskWrite)
	}

	// Validate we have at least CPU and memory (compute metrics are always required)
	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return correlationMetrics{}, fmt.Errorf("no compute metrics available for correlation analysis")
	}

	return metrics, nil
}

// Returns the data points for a given metric name
func getMetricData(metrics correlationMetrics, name string) []timeseries.DataPoint {
	switch name {
	case "cpu":
		return metrics.CPU
	case "memory":
		return metrics.Memory
	case "network_receive":
		return metrics.NetworkReceive
	case "network_transmit":
		return metrics.NetworkTransmit
	case "disk_read":
		return metrics.DiskRead
	case "disk_write":
		return metrics.DiskWrite
	default:
		return nil
	}
}

// Generates a human-readable interpretation of a correlation result
func interpretCorrelation(pair MetricPair, result timeseries.CrossCorrelationResult) string {
	coeff := result.MaxCorrelation.Coefficient
	strength := result.MaxCorrelation.Strength
	lag := result.OptimalLag

	if strength == timeseries.CorrelationStrengthWeak {
		return fmt.Sprintf("Weak correlation between %s and %s (r=%.2f)",
			formatMetricName(pair.MetricA), formatMetricName(pair.MetricB), coeff)
	}

	direction := timeseries.CorrelationDirectionPositive
	if coeff < 0 {
		direction = timeseries.CorrelationDirectionNegative
	}

	lagStr := ""
	if lag != 0 && result.LeadingSeries != "" {
		leadingMetric := pair.MetricA
		if result.LeadingSeries == "B" {
			leadingMetric = pair.MetricB
		}
		lagStr = fmt.Sprintf(" with %s leading by %v", formatMetricName(leadingMetric), absD(lag))
	}

	return fmt.Sprintf("%s %s correlation between %s and %s (r=%.2f)%s",
		cases.Title(language.English).String(string(strength)), direction,
		formatMetricName(pair.MetricA), formatMetricName(pair.MetricB),
		coeff, lagStr)
}

// Characterizes workload type based on correlation patterns
func characterizeWorkload(correlations []PairCorrelation) WorkloadType {
	// Extract key correlations
	var cpuMemory, cpuNetRecv, cpuNetTrans, cpuDiskRead, cpuDiskWrite float64
	var memNetRecv, netRecvDiskWrite float64

	for _, c := range correlations {
		if !c.DataAvailable {
			continue
		}
		coeff := c.Correlation.MaxCorrelation.Coefficient
		key := c.Pair.MetricA + "_" + c.Pair.MetricB

		switch key {
		case "cpu_memory":
			cpuMemory = coeff
		case "cpu_network_receive":
			cpuNetRecv = coeff
		case "cpu_network_transmit":
			cpuNetTrans = coeff
		case "cpu_disk_read":
			cpuDiskRead = coeff
		case "cpu_disk_write":
			cpuDiskWrite = coeff
		case "memory_network_receive":
			memNetRecv = coeff
		case "network_receive_disk_write":
			netRecvDiskWrite = coeff
		}
	}

	// Characterization rules
	// High CPU↔Memory with low I/O correlations -> Compute-bound
	if cpuMemory > 0.7 && abs(cpuNetRecv) < 0.3 && abs(cpuDiskRead) < 0.3 {
		return WorkloadTypeComputeBound
	}

	// High CPU↔Network correlations -> Request-driven
	if cpuNetRecv > 0.5 || cpuNetTrans > 0.5 {
		return WorkloadTypeRequestDriven
	}

	// High CPU↔Disk correlations -> Data-processing
	if cpuDiskRead > 0.5 || cpuDiskWrite > 0.5 {
		return WorkloadTypeDataProcessing
	}

	// High Network↔Disk correlation -> Pass-through
	if netRecvDiskWrite > 0.5 {
		return WorkloadTypePassThrough
	}

	// High memory, low CPU correlation -> Cache-heavy
	if abs(cpuMemory) < 0.3 && abs(memNetRecv) > 0.3 {
		return WorkloadTypeCacheHeavy
	}

	return WorkloadTypeMixed
}

// Generates insights based on correlation patterns
func generateInsights(correlations []PairCorrelation) []string {
	insights := make([]string, 0)

	// Track whether any moderate/strong correlations exist at all
	hasStrong := false
	hasModerate := false

	for _, c := range correlations {
		if !c.DataAvailable {
			continue
		}

		coeff := c.Correlation.MaxCorrelation.Coefficient
		strength := c.Correlation.MaxCorrelation.Strength
		lag := c.Correlation.OptimalLag

		// Only generate insights for moderate or strong correlations
		switch strength {
		case timeseries.CorrelationStrengthStrong:
			hasStrong = true
		case timeseries.CorrelationStrengthModerate:
			hasModerate = true
		default:
			continue
		}

		key := c.Pair.MetricA + "_" + c.Pair.MetricB

		switch key {
		case "cpu_memory":
			if coeff > 0.7 {
				insights = append(insights, "CPU and memory usage are tightly coupled, indicating the workload scales both resources together")
			} else if coeff < -0.5 {
				insights = append(insights, "CPU and memory show inverse relationship, possibly indicating memory caching reduces CPU work")
			}

		case "cpu_network_receive":
			if coeff > 0.5 {
				insights = append(insights, "CPU usage correlates with incoming network traffic, suggesting a request-driven workload")
				if lag != 0 && c.Correlation.LeadingSeries == "B" {
					insights = append(insights, fmt.Sprintf("Network traffic leads CPU by %v, indicating request processing delay", absD(lag)))
				}
			}

		case "cpu_disk_read":
			if coeff > 0.5 {
				insights = append(insights, "CPU usage correlates with disk reads, suggesting data processing workload")
			} else if coeff < -0.5 {
				insights = append(insights, "Inverse CPU-disk read correlation may indicate effective caching")
			}

		case "cpu_disk_write":
			if coeff > 0.5 {
				insights = append(insights, "CPU usage correlates with disk writes, consider async or buffered I/O if write latency is a concern")
			}

		case "memory_network_receive":
			if coeff > 0.5 {
				insights = append(insights, "Memory usage correlates with incoming traffic, indicating data buffering")
			}

		case "memory_disk_write":
			if coeff > 0.5 {
				insights = append(insights, "Memory and disk writes are correlated, which may indicate dirty page flush under memory pressure")
			} else if coeff < -0.5 {
				insights = append(insights, "Inverse memory-disk write correlation suggests memory caching reduces disk I/O")
			}

		case "network_receive_network_transmit":
			if coeff > 0.7 {
				insights = append(insights, "Incoming and outgoing network traffic are tightly coupled, indicating mostly symmetric network usage patterns")
			}

		case "network_receive_disk_write":
			if coeff > 0.5 {
				insights = append(insights, "Incoming network traffic correlates with disk writes, indicating pass-through or logging behavior")
			}

		case "disk_read_disk_write":
			if coeff > 0.7 {
				insights = append(insights, "Disk reads and writes are tightly coupled, indicating data transformation or copy operations")
			} else if coeff < -0.5 {
				insights = append(insights, "Inverse disk read/write correlation suggests alternating read-heavy and write-heavy phases")
			}
		}
	}

	if len(insights) == 0 {
		if hasStrong {
			insights = append(insights, "Strong correlations detected between some resources, but no predefined insight rules matched these patterns")
		} else if hasModerate {
			insights = append(insights, "Moderate correlations detected between some resources, but no predefined insight rules matched these patterns")
		} else {
			insights = append(insights, "No moderate or strong correlations detected, workload shows weakly coupled resource usage")
		}
	}

	return insights
}

// Generates optimization hints based on workload characterization
func generateOptimizationHints(workloadType WorkloadType, correlations []PairCorrelation) []string {
	hints := make([]string, 0)

	switch workloadType {
	case WorkloadTypeComputeBound:
		hints = append(hints, "Scale CPU and memory requests together based on observed correlation")
		hints = append(hints, "Consider balanced instance types that provide proportional CPU and memory")
		hints = append(hints, "Monitor both CPU throttling and memory pressure as both resources are tightly coupled")

	case WorkloadTypeRequestDriven:
		hints = append(hints, "Consider using Horizontal Pod Autoscaler (HPA) with network or request-based metrics")
		hints = append(hints, "Ensure network policies don't bottleneck incoming traffic")
		hints = append(hints, "Monitor request latency alongside resource usage for capacity planning")

	case WorkloadTypeDataProcessing:
		hints = append(hints, "Consider using faster storage classes (SSD) if disk I/O is a bottleneck")
		hints = append(hints, "Evaluate batch processing windows to avoid I/O contention with other workloads")
		hints = append(hints, "Monitor disk IOPS limits if using cloud storage with provisioned performance")

	case WorkloadTypePassThrough:
		hints = append(hints, "Scale network and storage resources together")
		hints = append(hints, "Consider using local storage or high-throughput network storage")
		hints = append(hints, "Monitor for network or disk saturation during peak traffic")

	case WorkloadTypeCacheHeavy:
		hints = append(hints, "Memory is the primary scaling dimension for this workload")
		hints = append(hints, "Ensure memory limits are set appropriately to prevent OOM kills")
		hints = append(hints, "Consider memory-optimized instance types if running on cloud infrastructure")

	case WorkloadTypeMixed:
		hints = append(hints, "Workload shows diverse resource patterns, analyze individual metrics for optimization")
		hints = append(hints, "Consider breaking down into sub-components if resource usage varies significantly")
	}

	// Add specific hints based on correlation patterns
	for _, c := range correlations {
		if !c.DataAvailable {
			continue
		}

		coeff := c.Correlation.MaxCorrelation.Coefficient
		key := c.Pair.MetricA + "_" + c.Pair.MetricB

		switch key {
		case "network_receive_network_transmit":
			if abs(coeff) < 0.3 {
				hints = append(hints, "Asymmetric network traffic detected, consider separate ingress/egress scaling strategies")
			}

		case "disk_read_disk_write":
			if coeff < -0.5 {
				hints = append(hints, "Alternating read/write patterns detected, consider separating read and write paths if possible")
			} else if abs(coeff) < 0.3 {
				hints = append(hints, "Independent read/write patterns, may benefit from separate read and write optimization strategies")
			}
		}
	}

	return hints
}

// Formats a metric name for display
func formatMetricName(name string) string {
	switch name {
	case "cpu":
		return "CPU"
	case "memory":
		return "Memory"
	case "network_receive":
		return "Network Receive"
	case "network_transmit":
		return "Network Transmit"
	case "disk_read":
		return "Disk Read"
	case "disk_write":
		return "Disk Write"
	default:
		return name
	}
}

// Returns absolute value of float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Returns absolute value of duration
func absD(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
