package compute

import (
	"github.com/thread_koder/mochi/core/internal/timeseries"
)

// ResourceSpecs is parsed Kubernetes CPU (cores) and memory (bytes) requests and limits.
type ResourceSpecs struct {
	CPURequest    *float64 `json:"cpu_request"`
	CPULimit      *float64 `json:"cpu_limit"`
	MemoryRequest *float64 `json:"memory_request"`
	MemoryLimit   *float64 `json:"memory_limit"`
}

type CPUProvisioning struct {
	RequestUtilization float64  `json:"request_utilization"`
	LimitUtilization   float64  `json:"limit_utilization"`
	CurrentRequest     *float64 `json:"current_request"`
	CurrentLimit       *float64 `json:"current_limit"`
	IsOverProvisioned  bool     `json:"is_over_provisioned"`
	IsUnderProvisioned bool     `json:"is_under_provisioned"`
	Efficiency         float64  `json:"efficiency"`
	Confidence         float64  `json:"confidence"`
}

type MemoryProvisioning struct {
	RequestUtilization float64  `json:"request_utilization"`
	LimitUtilization   float64  `json:"limit_utilization"`
	CurrentRequest     *float64 `json:"current_request"`
	CurrentLimit       *float64 `json:"current_limit"`
	IsOverProvisioned  bool     `json:"is_over_provisioned"`
	IsUnderProvisioned bool     `json:"is_under_provisioned"`
	Efficiency         float64  `json:"efficiency"`
	Confidence         float64  `json:"confidence"`
}

type ProvisioningResult struct {
	CPU        CPUProvisioning    `json:"cpu"`
	Memory     MemoryProvisioning `json:"memory"`
	Efficiency float64            `json:"efficiency"`
}

const (
	OptimalUtilizationMin = 0.4
	OptimalUtilizationMax = 0.7

	CPUHeadroom    = 0.2
	MemoryHeadroom = 0.2

	// Used to skip "over provisioned" flags so we do not nag on minimum resources.
	MinCPURequestCores    = 0.01
	MinMemoryRequestBytes = 64 * 1024 * 1024

	ThrottlingThreshold = 0.05
	PressureThreshold   = 0.1

	BurstEffectiveMinFloor = 0.05
	BurstEffectiveMinCeil  = 0.4

	// Confidence burst softener: partial predictability credit when usage is bursty but well observed.
	confidenceBurstThreshold  = 1.6
	confidenceBurstFloor      = 0.8
	confidenceDataFactorFloor = 0.5
)

func AnalyzeCPUProvisioning(specs ResourceSpecs, utilization CPUUtilization, stability StabilityResult, minSamples int) CPUProvisioning {
	result := CPUProvisioning{
		IsOverProvisioned:  false,
		IsUnderProvisioned: false,
		Efficiency:         1.0,
		Confidence: computeResourceConfidence(
			utilization.Stats,
			utilization.SampleSize,
			minSamples,
		),
	}

	hasRequest := specs.CPURequest != nil && *specs.CPURequest > 0
	hasLimit := specs.CPULimit != nil && *specs.CPULimit > 0

	result.CurrentRequest = specs.CPURequest
	result.CurrentLimit = specs.CPULimit

	if !hasRequest || !hasLimit {
		result.IsUnderProvisioned = true
	}

	if !hasRequest && !hasLimit {
		result.Efficiency = 0.0
		return result
	}

	if stability.CPUThrottling > ThrottlingThreshold {
		result.IsUnderProvisioned = true
		penalty := (stability.CPUThrottling - ThrottlingThreshold) * 3.0
		result.Efficiency = min(result.Efficiency, max(0.0, 1.0-penalty))
	}
	if stability.CPUPressure > PressureThreshold {
		result.IsUnderProvisioned = true
		penalty := (stability.CPUPressure - PressureThreshold) * 1.0
		result.Efficiency = min(result.Efficiency, max(0.0, 1.0-penalty))
	}

	if hasRequest {
		result.RequestUtilization = utilization.Stats.Percentile.P95 / *specs.CPURequest

		minThreshold := effectiveMinFromBurstiness(
			utilization.Stats.Mean,
			utilization.Stats.Percentile.P95,
			utilization.Stats.Max,
		)
		atFloor := *specs.CPURequest <= MinCPURequestCores
		lowThrottling := stability.CPUThrottling > 0 && stability.CPUThrottling <= ThrottlingThreshold
		underProvisionedDueToStability := stability.CPUThrottling > ThrottlingThreshold || stability.CPUPressure > PressureThreshold

		if result.RequestUtilization < minThreshold && !atFloor && !lowThrottling && !underProvisionedDueToStability {
			result.IsOverProvisioned = true
		}

		if result.RequestUtilization > OptimalUtilizationMax {
			result.IsUnderProvisioned = true
		}

		var requestEfficiency float64
		if result.RequestUtilization >= minThreshold && result.RequestUtilization <= OptimalUtilizationMax {
			requestEfficiency = 1.0
		} else if atFloor || lowThrottling {
			requestEfficiency = 1.0
		} else if result.RequestUtilization < minThreshold {
			requestEfficiency = result.RequestUtilization / minThreshold
		} else {
			if result.RequestUtilization > 1.0 {
				requestEfficiency = 0.0
			} else {
				requestEfficiency = 1.0 - ((result.RequestUtilization - OptimalUtilizationMax) / (1.0 - OptimalUtilizationMax))
			}
		}
		result.Efficiency = min(result.Efficiency, requestEfficiency)
	}

	if hasLimit {
		result.LimitUtilization = utilization.Stats.Max / *specs.CPULimit

		if result.LimitUtilization > (1.0 - CPUHeadroom) {
			result.IsUnderProvisioned = true
			limitPenalty := 1.0
			if result.LimitUtilization > 1.0 {
				limitPenalty = 0.0
			} else {
				limitPenalty = (1.0 - result.LimitUtilization) / CPUHeadroom
			}
			result.Efficiency = min(result.Efficiency, limitPenalty)
		}
	}

	if (hasRequest && !hasLimit) || (!hasRequest && hasLimit) {
		result.Efficiency = min(result.Efficiency, 0.5)
	}

	result.Efficiency = max(0.0, min(1.0, result.Efficiency))

	return result
}

func AnalyzeMemoryProvisioning(specs ResourceSpecs, utilization MemoryUtilization, stability StabilityResult, minSamples int) MemoryProvisioning {
	result := MemoryProvisioning{
		IsOverProvisioned:  false,
		IsUnderProvisioned: false,
		Efficiency:         1.0,
		Confidence: computeResourceConfidence(
			utilization.Stats,
			utilization.SampleSize,
			minSamples,
		),
	}

	hasRequest := specs.MemoryRequest != nil && *specs.MemoryRequest > 0
	hasLimit := specs.MemoryLimit != nil && *specs.MemoryLimit > 0

	result.CurrentRequest = specs.MemoryRequest
	result.CurrentLimit = specs.MemoryLimit

	if !hasRequest || !hasLimit {
		result.IsUnderProvisioned = true
	}

	if !hasRequest && !hasLimit {
		result.Efficiency = 0.0
		return result
	}

	if stability.MemoryOOM > 0 {
		result.IsUnderProvisioned = true
		penalty := eventCountPenalty(stability.MemoryOOM, maxPenaltyOOM, oomCountAtMax)
		result.Efficiency = min(result.Efficiency, max(0.0, 1.0-penalty))
	}
	if stability.MemoryFailCnt > 0 {
		result.IsUnderProvisioned = true
		penalty := eventCountPenalty(stability.MemoryFailCnt, maxPenaltyMemoryFailCnt, memoryFailCountAtMax)
		result.Efficiency = min(result.Efficiency, max(0.0, 1.0-penalty))
	}
	if stability.MemoryPressure > PressureThreshold {
		result.IsUnderProvisioned = true
		penalty := (stability.MemoryPressure - PressureThreshold) * 1.0
		result.Efficiency = min(result.Efficiency, max(0.0, 1.0-penalty))
	}

	if hasRequest {
		result.RequestUtilization = utilization.Stats.Percentile.P95 / *specs.MemoryRequest

		atFloor := *specs.MemoryRequest <= MinMemoryRequestBytes
		lowMemoryPressure := stability.MemoryPressure > 0 && stability.MemoryPressure <= PressureThreshold
		underProvisionedDueToStability := stability.MemoryOOM > 0 || stability.MemoryFailCnt > 0 || stability.MemoryPressure > PressureThreshold

		if result.RequestUtilization < OptimalUtilizationMin && !atFloor && !lowMemoryPressure && !underProvisionedDueToStability {
			result.IsOverProvisioned = true
		}

		if result.RequestUtilization > OptimalUtilizationMax {
			result.IsUnderProvisioned = true
		}

		var requestEfficiency float64
		if result.RequestUtilization >= OptimalUtilizationMin && result.RequestUtilization <= OptimalUtilizationMax {
			requestEfficiency = 1.0
		} else if atFloor || lowMemoryPressure {
			requestEfficiency = 1.0
		} else if result.RequestUtilization < OptimalUtilizationMin {
			requestEfficiency = result.RequestUtilization / OptimalUtilizationMin
		} else {
			if result.RequestUtilization > 1.0 {
				requestEfficiency = 0.0
			} else {
				requestEfficiency = 1.0 - ((result.RequestUtilization - OptimalUtilizationMax) / (1.0 - OptimalUtilizationMax))
			}
		}
		result.Efficiency = min(result.Efficiency, requestEfficiency)
	}

	if hasLimit {
		result.LimitUtilization = utilization.Stats.Max / *specs.MemoryLimit

		if result.LimitUtilization > (1.0 - MemoryHeadroom) {
			result.IsUnderProvisioned = true
			limitPenalty := 1.0
			if result.LimitUtilization > 1.0 {
				limitPenalty = 0.0
			} else {
				limitPenalty = (1.0 - result.LimitUtilization) / MemoryHeadroom
			}
			result.Efficiency = min(result.Efficiency, limitPenalty)
		}
	}

	if (hasRequest && !hasLimit) || (!hasRequest && hasLimit) {
		result.Efficiency = min(result.Efficiency, 0.5)
	}

	result.Efficiency = max(0.0, min(1.0, result.Efficiency))

	return result
}

func AnalyzeProvisioning(specs ResourceSpecs, utilization UtilizationResult, stability StabilityResult, minSamples int) ProvisioningResult {
	result := ProvisioningResult{
		CPU:    AnalyzeCPUProvisioning(specs, utilization.CPU, stability, minSamples),
		Memory: AnalyzeMemoryProvisioning(specs, utilization.Memory, stability, minSamples),
	}

	const cpuWeight = 0.3
	const memoryWeight = 0.7
	result.Efficiency = (result.CPU.Efficiency * cpuWeight) + (result.Memory.Efficiency * memoryWeight)

	return result
}

// computeResourceConfidence scores measurement trust from usage predictability and data sufficiency.
// Bursty workloads with enough samples receive a predictability floor so cron-style patterns are not
// over-penalized by coefficient of variation alone.
func computeResourceConfidence(stats timeseries.StatsResult, sampleSize, minSamples int) float64 {
	if stats.Mean == 0 {
		return 0
	}

	var predictability float64
	if stats.StdDev > 0 {
		cv := stats.StdDev / stats.Mean
		predictability = min(1.0, 1.0/(1.0+cv))
	} else {
		predictability = 1.0
	}

	if minSamples > 0 && sampleSize >= minSamples {
		score := burstScore(stats.Mean, stats.Percentile.P95, stats.Max)
		if score > confidenceBurstThreshold {
			predictability = max(predictability, confidenceBurstFloor)
		}
	}

	if minSamples <= 0 {
		return max(0.0, min(1.0, predictability))
	}

	dataFactor := min(1.0, float64(sampleSize)/float64(minSamples))
	if dataFactor < confidenceDataFactorFloor {
		return 0
	}

	return max(0.0, min(1.0, predictability*dataFactor))
}

func burstScore(mean, p95, peak float64) float64 {
	if mean <= 0 {
		return 0
	}
	return (p95/mean + peak/mean) / 2.0
}

// effectiveMinFromBurstiness lowers the minimum "healthy" request utilization when mean, P95, and peak
// show burstiness, so bursty CPU workloads are not marked over-provisioned for sitting near idle between spikes.
func effectiveMinFromBurstiness(mean, p95, peak float64) float64 {
	if mean <= 0 {
		return BurstEffectiveMinCeil
	}
	score := max(burstScore(mean, p95, peak), 1.0)
	effectiveMin := BurstEffectiveMinCeil - (score-1.0)*0.1
	effectiveMin = max(effectiveMin, BurstEffectiveMinFloor)
	effectiveMin = min(effectiveMin, BurstEffectiveMinCeil)
	return effectiveMin
}
