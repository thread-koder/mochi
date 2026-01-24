package compute

import (
	"math"
)

// Represents stability analysis results for a container
type StabilityResult struct {
	CPUThrottling  float64 `json:"cpu_throttling"`  // Average throttling rate (periods/sec)
	CPUPressure    float64 `json:"cpu_pressure"`    // Average CPU pressure (seconds/sec)
	MemoryFailCnt  float64 `json:"memory_fail_cnt"` // Memory allocation failures
	MemoryOOM      float64 `json:"memory_oom"`      // OOM events
	MemoryPressure float64 `json:"memory_pressure"` // Average memory pressure (seconds/sec)
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
		result.MemoryFailCnt = CalculateMean(ExtractValues(metrics.MemoryFailCnt))
	}
	if len(metrics.MemoryOOM) > 0 {
		// Sum of OOM events
		for _, dp := range metrics.MemoryOOM {
			result.MemoryOOM += dp.Value
		}
	}
	if len(metrics.MemoryPressure) > 0 {
		result.MemoryPressure = CalculateMean(ExtractValues(metrics.MemoryPressure))
	}
	if len(metrics.Restarts) > 0 {
		// Sum of restarts
		for _, dp := range metrics.Restarts {
			result.Restarts += dp.Value
		}
	}

	// 2. Calculate Stability Score (0-1)

	// OOMs are critical
	result.StabilityScore -= result.MemoryOOM * 0.5

	// Restarts are critical
	result.StabilityScore -= result.Restarts * 0.3

	// Throttling: penalize if > 10%
	if result.CPUThrottling > 0.1 {
		result.StabilityScore -= (result.CPUThrottling - 0.1) * 2.0
	}

	// Pressure: penalize if > 20% stalled
	if result.CPUPressure > 0.2 {
		result.StabilityScore -= (result.CPUPressure - 0.2) * 0.5
	}
	if result.MemoryPressure > 0.2 {
		result.StabilityScore -= (result.MemoryPressure - 0.2) * 1.0
	}

	// Clamp score to 0-1
	result.StabilityScore = math.Max(0.0, math.Min(1.0, result.StabilityScore))

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

	var totalScore float64
	for _, s := range stabilities {
		aggregated.CPUThrottling += s.CPUThrottling
		aggregated.CPUPressure += s.CPUPressure
		aggregated.MemoryFailCnt += s.MemoryFailCnt
		aggregated.MemoryOOM += s.MemoryOOM
		aggregated.MemoryPressure += s.MemoryPressure
		aggregated.Restarts += s.Restarts
		totalScore += s.StabilityScore
	}

	// For rates/counts, sum or average depending on meaning
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
