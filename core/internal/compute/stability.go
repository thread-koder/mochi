package compute

import "math"

// Per-signal caps on how much each factor can reduce StabilityScore. Without caps, a single noisy
// counter could drive the score to zero.
const (
	maxPenaltyCPUThrottling  = 0.25
	maxPenaltyCPUPressure    = 0.15
	maxPenaltyMemoryPressure = 0.15

	// Log-scaled event penalties.
	maxPenaltyOOM           = 0.32
	oomCountAtMax           = 1
	maxPenaltyMemoryFailCnt = 0.22
	memoryFailCountAtMax    = 10
	maxPenaltyRestarts      = 0.25
	restartsCountAtMax      = 25
)

// stabilityNoiseThreshold treats PSI and throttling values below 0.1% as zero so scrape jitter
// does not look like real pressure.
const stabilityNoiseThreshold = 0.001

// StabilityResult holds raw stability signals (as fractions or event counts from Prometheus) and
// a combined StabilityScore in [0,1].
type StabilityResult struct {
	CPUThrottling  float64 `json:"cpu_throttling"`
	CPUPressure    float64 `json:"cpu_pressure"`
	MemoryFailCnt  float64 `json:"memory_fail_cnt"`
	MemoryOOM      float64 `json:"memory_oom"`
	MemoryPressure float64 `json:"memory_pressure"`
	Restarts       float64 `json:"restarts"`
	StabilityScore float64 `json:"stability_score"`
}

// AnalyzeStability reads the first sample in each scalar slice (see ResourceMetrics), applies
// filterNoise to percentage fields, and subtracts capped penalties from 1.0. OOM, allocation
// failures, and restarts use log-scaled event penalties, while throttling above 5% and CPU/memory
// pressure above 10% use linear ramps.
func AnalyzeStability(metrics ResourceMetrics) StabilityResult {
	result := StabilityResult{
		StabilityScore: 1.0,
	}

	if len(metrics.CPUThrottling) > 0 {
		result.CPUThrottling = metrics.CPUThrottling[0].Value
	}
	if len(metrics.CPUPressure) > 0 {
		result.CPUPressure = metrics.CPUPressure[0].Value
	}
	if len(metrics.MemoryFailCnt) > 0 {
		result.MemoryFailCnt = math.Round(metrics.MemoryFailCnt[0].Value)
	}
	if len(metrics.MemoryOOM) > 0 {
		result.MemoryOOM = math.Round(metrics.MemoryOOM[0].Value)
	}
	if len(metrics.MemoryPressure) > 0 {
		result.MemoryPressure = metrics.MemoryPressure[0].Value
	}
	if len(metrics.Restarts) > 0 {
		result.Restarts = math.Round(metrics.Restarts[0].Value)
	}

	result.CPUThrottling = filterNoise(result.CPUThrottling)
	result.CPUPressure = filterNoise(result.CPUPressure)
	result.MemoryPressure = filterNoise(result.MemoryPressure)

	var totalPenalty float64

	totalPenalty += eventCountPenalty(result.MemoryOOM, maxPenaltyOOM, oomCountAtMax)
	totalPenalty += eventCountPenalty(result.MemoryFailCnt, maxPenaltyMemoryFailCnt, memoryFailCountAtMax)
	totalPenalty += eventCountPenalty(result.Restarts, maxPenaltyRestarts, restartsCountAtMax)

	if result.CPUThrottling > 0.05 {
		penalty := (result.CPUThrottling - 0.05) * 2.0
		totalPenalty += min(penalty, maxPenaltyCPUThrottling)
	}

	if result.CPUPressure > 0.1 {
		penalty := (result.CPUPressure - 0.1) * 0.5
		totalPenalty += min(penalty, maxPenaltyCPUPressure)
	}
	if result.MemoryPressure > 0.1 {
		penalty := (result.MemoryPressure - 0.1) * 1.0
		totalPenalty += min(penalty, maxPenaltyMemoryPressure)
	}

	result.StabilityScore = max(0.0, min(1.0, 1.0-totalPenalty))

	return result
}

// eventCountPenalty maps discrete counter events to a capped stability penalty. Log scaling
// keeps single events as a warning while higher counts increase severity sublinearly.
func eventCountPenalty(count, maxPenalty, countAtMax float64) float64 {
	if count <= 0 || maxPenalty <= 0 || countAtMax <= 0 {
		return 0
	}
	penalty := maxPenalty * math.Log1p(count) / math.Log1p(countAtMax)
	return min(penalty, maxPenalty)
}

// filterNoise returns zero when value is below stabilityNoiseThreshold, otherwise returns value.
func filterNoise(value float64) float64 {
	if value < stabilityNoiseThreshold {
		return 0
	}
	return value
}
