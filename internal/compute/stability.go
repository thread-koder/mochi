package compute

import (
	"math"
)

// Max penalty per factor
const (
	maxPenaltyCPUThrottling  = 0.25 // 25% max penalty
	maxPenaltyCPUPressure    = 0.15 // 15% max penalty
	maxPenaltyMemoryFailCnt  = 0.25 // 25% max penalty
	maxPenaltyMemoryPressure = 0.15 // 15% max penalty
	maxPenaltyOOM            = 0.35 // 35% max penalty
	maxPenaltyRestarts       = 0.35 // 35% max penalty
)

// Represents stability analysis results for a container
type StabilityResult struct {
	CPUThrottling  float64 `json:"cpu_throttling"`  // CPU throttling percentage (0-1, where 1.0 = 100% throttling)
	CPUPressure    float64 `json:"cpu_pressure"`    // CPU pressure percentage (0-1, where 1.0 = 100% stalled)
	MemoryFailCnt  float64 `json:"memory_fail_cnt"` // Total memory allocation failures in period
	MemoryOOM      float64 `json:"memory_oom"`      // Total OOM events in period
	MemoryPressure float64 `json:"memory_pressure"` // Memory pressure percentage (0-1, where 1.0 = 100% stalled)
	Restarts       float64 `json:"restarts"`        // Total restarts in period
	StabilityScore float64 `json:"stability_score"` // Overall stability score (0-1)
}

// Analyzes stability from metrics
func AnalyzeStability(metrics ResourceMetrics) (StabilityResult, error) {
	result := StabilityResult{
		StabilityScore: 1.0, // Start at optimal then penalize
	}

	// 1. Calculate raw metrics
	if len(metrics.CPUThrottling) > 0 {
		result.CPUThrottling = CalculateMean(ExtractValues(metrics.CPUThrottling))
	}
	if len(metrics.CPUPressure) > 0 {
		result.CPUPressure = CalculateMean(ExtractValues(metrics.CPUPressure))
	}
	if len(metrics.MemoryFailCnt) > 0 {
		values := ExtractValues(metrics.MemoryFailCnt)
		result.MemoryFailCnt = values[len(values)-1]
	}
	if len(metrics.MemoryOOM) > 0 {
		values := ExtractValues(metrics.MemoryOOM)
		result.MemoryOOM = values[len(values)-1]
	}
	if len(metrics.MemoryPressure) > 0 {
		result.MemoryPressure = CalculateMean(ExtractValues(metrics.MemoryPressure))
	}
	if len(metrics.Restarts) > 0 {
		values := ExtractValues(metrics.Restarts)
		result.Restarts = values[len(values)-1]
	}

	// 2. Calculate Stability Score (0-1)
	var totalPenalty float64

	// OOMs: critical
	totalPenalty += math.Min(result.MemoryOOM*0.5, maxPenaltyOOM)

	// Memory allocation failures: pre-OOM signal
	totalPenalty += math.Min(result.MemoryFailCnt*0.2, maxPenaltyMemoryFailCnt)

	// Restarts: critical
	totalPenalty += math.Min(result.Restarts*0.3, maxPenaltyRestarts)

	// Throttling: penalize if > 10%
	if result.CPUThrottling > 0.1 {
		penalty := (result.CPUThrottling - 0.1) * 2.0
		totalPenalty += math.Min(penalty, maxPenaltyCPUThrottling)
	}

	// Pressure: penalize if > 20% stalled
	if result.CPUPressure > 0.2 {
		penalty := (result.CPUPressure - 0.2) * 0.5
		totalPenalty += math.Min(penalty, maxPenaltyCPUPressure)
	}
	if result.MemoryPressure > 0.2 {
		penalty := (result.MemoryPressure - 0.2) * 1.0
		totalPenalty += math.Min(penalty, maxPenaltyMemoryPressure)
	}

	result.StabilityScore = math.Max(0.0, math.Min(1.0, 1.0-totalPenalty))

	return result, nil
}

// Aggregates stability results from multiple containers into a single result (Pod or Workload level)
func AggregateStability(stabilities []StabilityResult) StabilityResult {
	if len(stabilities) == 0 {
		return StabilityResult{StabilityScore: 1.0}
	}

	aggregated := StabilityResult{
		StabilityScore: 1.0,
	}

	for _, s := range stabilities {
		aggregated.CPUThrottling += s.CPUThrottling
		aggregated.CPUPressure += s.CPUPressure
		aggregated.MemoryFailCnt += s.MemoryFailCnt
		aggregated.MemoryOOM += s.MemoryOOM
		aggregated.MemoryPressure += s.MemoryPressure
		aggregated.Restarts += s.Restarts
	}

	// Average percentages (CPU throttling, CPU pressure, memory pressure)
	n := float64(len(stabilities))
	aggregated.CPUThrottling /= n
	aggregated.CPUPressure /= n
	aggregated.MemoryPressure /= n

	// Stability score is the minimum
	minScore := 1.0
	for _, s := range stabilities {
		if s.StabilityScore < minScore {
			minScore = s.StabilityScore
		}
	}
	aggregated.StabilityScore = minScore

	return aggregated
}
