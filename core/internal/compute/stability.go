package compute

// Per-signal caps on how much each factor can reduce StabilityScore. Without caps, a single noisy
// counter could drive the score to zero.
const (
	maxPenaltyCPUThrottling  = 0.25
	maxPenaltyCPUPressure    = 0.15
	maxPenaltyMemoryFailCnt  = 0.25
	maxPenaltyMemoryPressure = 0.15
	maxPenaltyOOM            = 0.35
	maxPenaltyRestarts       = 0.35
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
// filterNoise to percentage fields, and subtracts capped penalties from 1.0 for OOMs, allocation
// failures, restarts, throttling above 5%, and CPU/memory pressure above 10%.
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
		result.MemoryFailCnt = metrics.MemoryFailCnt[0].Value
	}
	if len(metrics.MemoryOOM) > 0 {
		result.MemoryOOM = metrics.MemoryOOM[0].Value
	}
	if len(metrics.MemoryPressure) > 0 {
		result.MemoryPressure = metrics.MemoryPressure[0].Value
	}
	if len(metrics.Restarts) > 0 {
		result.Restarts = metrics.Restarts[0].Value
	}

	result.CPUThrottling = filterNoise(result.CPUThrottling)
	result.CPUPressure = filterNoise(result.CPUPressure)
	result.MemoryPressure = filterNoise(result.MemoryPressure)

	var totalPenalty float64

	totalPenalty += min(result.MemoryOOM*0.5, maxPenaltyOOM)
	totalPenalty += min(result.MemoryFailCnt*0.2, maxPenaltyMemoryFailCnt)
	totalPenalty += min(result.Restarts*0.3, maxPenaltyRestarts)

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

// filterNoise returns zero when value is below stabilityNoiseThreshold, otherwise returns value.
func filterNoise(value float64) float64 {
	if value < stabilityNoiseThreshold {
		return 0
	}
	return value
}
