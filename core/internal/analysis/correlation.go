package analysis

import (
	"context"
	"fmt"
	"time"

	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/timeseries"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// CorrelationOptions controls the analysis window and lag search bounds.
type CorrelationOptions struct {
	TimeRange time.Duration // How far back to analyze (default: 24h).
	RangeStep time.Duration // Step size for Prometheus range queries (default: 1m).
	MaxLag    time.Duration // Largest lag checked during cross-correlation (default: 5m).
	LagStep   time.Duration // Lag increment used while searching for MaxLag (default: 1m).
}

// DefaultCorrelationOptions returns a 24 hour window with a 1 minute range step,
// a 5 minute max lag, and a 1 minute lag step.
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

// SetTimeRange stores timeRange, sets RangeStep from tiered resolution rules, and syncs lag settings.
func (opts *CorrelationOptions) SetTimeRange(timeRange time.Duration) {
	opts.TimeRange = timeRange
	opts.RangeStep = timeseries.RangeStepForTimeRange(timeRange)
	opts.syncLagSettings()
}

// syncLagSettings aligns lag search resolution with RangeStep so cross-correlation
// does not probe sub-step offsets that the sampled data cannot support.
func (opts *CorrelationOptions) syncLagSettings() {
	opts.LagStep = max(opts.LagStep, opts.RangeStep)
	if opts.MaxLag > 0 {
		opts.MaxLag = max(opts.MaxLag, opts.LagStep)
	}
}

// Validate ensures that the correlation options are valid before analyzing.
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

// MetricPair identifies the two metrics to compare together.
type MetricPair struct {
	MetricA string `json:"metric_a"`
	MetricB string `json:"metric_b"`
}

// PairCorrelation contains the computed relationship for one MetricPair.
type PairCorrelation struct {
	Pair           MetricPair                        `json:"pair"`
	Correlation    timeseries.CrossCorrelationResult `json:"correlation"`
	Interpretation string                            `json:"interpretation"`
	DataAvailable  bool                              `json:"data_available"`
}

// WorkloadType describes the dominant resource pattern inferred from
// cross-metric correlations.
type WorkloadType string

const (
	WorkloadTypeComputeBound   WorkloadType = "compute-bound"
	WorkloadTypeRequestDriven  WorkloadType = "request-driven"
	WorkloadTypeDataProcessing WorkloadType = "data-processing"
	WorkloadTypePassThrough    WorkloadType = "pass-through"
	WorkloadTypeCacheHeavy     WorkloadType = "cache-heavy"
	WorkloadTypeMixed          WorkloadType = "mixed"
)

// WorkloadCorrelationResult is the final result for the workload-level
// cross-metric correlation analysis.
type WorkloadCorrelationResult struct {
	WorkloadType      string            `json:"workload_type"`
	WorkloadName      string            `json:"workload_name"`
	Namespace         string            `json:"namespace"`
	Correlations      []PairCorrelation `json:"correlations"`
	CharacterizedAs   WorkloadType      `json:"characterized_as"`
	Insights          []string          `json:"insights"`
	OptimizationHints []string          `json:"optimization_hints"`
	DataPointsUsed    int               `json:"data_points_used"`
}

// metricPairs defines a set of cross-metric comparisons to analyze.
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

// AnalyzeWorkloadCorrelations computes cross-metric correlations across compute,
// network, and disk metrics for one workload.
func AnalyzeWorkloadCorrelations(ctx context.Context, workloadType string, workloadName string, namespace string, pods []*database.Pod, opts CorrelationOptions) (WorkloadCorrelationResult, error) {
	if err := opts.Validate(); err != nil {
		return WorkloadCorrelationResult{}, fmt.Errorf("invalid correlation options: %w", err)
	}

	metrics, err := fetchWorkloadCorrelationMetrics(ctx, pods, opts)
	if err != nil {
		return WorkloadCorrelationResult{}, fmt.Errorf("failed to fetch workload metrics: %w", err)
	}

	// At least one of CPU or memory is required as it anchors
	// workload characterization.
	// Network and disk are best-effort as some pods may not expose those metrics.
	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return WorkloadCorrelationResult{}, apperrors.NewNoMetrics(fmt.Sprintf("workload %s/%s", namespace, workloadName))
	}

	correlations := make([]PairCorrelation, 0, len(metricPairs))

	maxDataPoints := 0
	for _, pair := range metricPairs {
		dataA := metricData(metrics, pair.MetricA)
		dataB := metricData(metrics, pair.MetricB)

		pairCorr := PairCorrelation{
			Pair:          pair,
			DataAvailable: len(dataA) >= 2 && len(dataB) >= 2,
		}

		if pairCorr.DataAvailable {
			crossCorr, err := timeseries.CalculateCrossCorrelation(dataA, dataB, opts.MaxLag, opts.LagStep)
			if err != nil {
				pairCorr.DataAvailable = false
				pairCorr.Interpretation = "Could not calculate correlation"
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

	characterization := characterizeWorkload(correlations)
	insights := generateInsights(correlations)
	optimizationHints := generateOptimizationHints(characterization, correlations)

	return WorkloadCorrelationResult{
		WorkloadType:      workloadType,
		WorkloadName:      workloadName,
		Namespace:         namespace,
		Correlations:      correlations,
		CharacterizedAs:   characterization,
		Insights:          insights,
		OptimizationHints: optimizationHints,
		DataPointsUsed:    maxDataPoints,
	}, nil
}

// metricData returns the series for a given metric name.
func metricData(metrics correlationMetrics, name string) []timeseries.DataPoint {
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

// interpretCorrelation summarizes strength, direction, and lag in plain text.
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
		lagStr = fmt.Sprintf(" with %s leading by %v", formatMetricName(leadingMetric), absDuration(lag))
	}

	return fmt.Sprintf("%s %s correlation between %s and %s (r=%.2f)%s",
		cases.Title(language.English).String(string(strength)), direction,
		formatMetricName(pair.MetricA), formatMetricName(pair.MetricB),
		coeff, lagStr)
}

// characterizeWorkload applies heuristic thresholds to label workload behavior.
func characterizeWorkload(correlations []PairCorrelation) WorkloadType {
	var cpuMemory, cpuNetRecv, cpuNetTrans, cpuDiskRead, cpuDiskWrite float64
	var memNetRecv, netRecvDiskWrite float64

	for _, c := range correlations {
		if !c.DataAvailable {
			continue
		}
		coeff := c.Correlation.MaxCorrelation.Coefficient
		pairKey := c.Pair.MetricA + "_" + c.Pair.MetricB

		switch pairKey {
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

	// CPU and memory moving together while I/O stays decoupled usually means
	// the workload saturates compute before network or storage.
	if cpuMemory > 0.7 && abs(cpuNetRecv) < 0.3 && abs(cpuDiskRead) < 0.3 {
		return WorkloadTypeComputeBound
	}

	// CPU tracking inbound or outbound traffic is a common request-driven shape.
	if cpuNetRecv > 0.5 || cpuNetTrans > 0.5 {
		return WorkloadTypeRequestDriven
	}

	// CPU following disk activity often indicates transform/ETL style work.
	if cpuDiskRead > 0.5 || cpuDiskWrite > 0.5 {
		return WorkloadTypeDataProcessing
	}

	// Network and disk moving together often reflects pass-through pipelines.
	if netRecvDiskWrite > 0.5 {
		return WorkloadTypePassThrough
	}

	// Memory movement without matching CPU pressure often maps to cache-heavy
	// access patterns.
	if abs(cpuMemory) < 0.3 && abs(memNetRecv) > 0.3 {
		return WorkloadTypeCacheHeavy
	}

	return WorkloadTypeMixed
}

// generateInsights generates human-readable insights for moderate/strong signals.
func generateInsights(correlations []PairCorrelation) []string {
	insights := make([]string, 0)

	hasStrong := false
	hasModerate := false

	for _, c := range correlations {
		if !c.DataAvailable {
			continue
		}

		coeff := c.Correlation.MaxCorrelation.Coefficient
		strength := c.Correlation.MaxCorrelation.Strength
		lag := c.Correlation.OptimalLag

		switch strength {
		case timeseries.CorrelationStrengthStrong:
			hasStrong = true
		case timeseries.CorrelationStrengthModerate:
			hasModerate = true
		default:
			continue
		}

		pairKey := c.Pair.MetricA + "_" + c.Pair.MetricB

		switch pairKey {
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
					insights = append(insights, fmt.Sprintf("Network traffic leads CPU by %v, indicating request processing delay", absDuration(lag)))
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

// generateOptimizationHints generates tuning suggestions from characterization
// plus a few targeted pair patterns.
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

	for _, c := range correlations {
		if !c.DataAvailable {
			continue
		}

		coeff := c.Correlation.MaxCorrelation.Coefficient
		pairKey := c.Pair.MetricA + "_" + c.Pair.MetricB

		switch pairKey {
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

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
