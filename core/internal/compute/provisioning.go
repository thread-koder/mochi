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

type ProvisioningStatus string

const (
	ProvisioningUnspecified      ProvisioningStatus = "unspecified"
	ProvisioningOverProvisioned  ProvisioningStatus = "over_provisioned"
	ProvisioningUnderProvisioned ProvisioningStatus = "under_provisioned"
	ProvisioningOptimal          ProvisioningStatus = "optimal"
)

type ResourceProvisioning struct {
	RequestUtilization *float64           `json:"request_utilization"`
	LimitUtilization   *float64           `json:"limit_utilization"`
	CurrentRequest     *float64           `json:"current_request"`
	CurrentLimit       *float64           `json:"current_limit"`
	Status             ProvisioningStatus `json:"status"`
	Efficiency         float64            `json:"efficiency"`
	Confidence         float64            `json:"confidence"`
}

type ProvisioningResult struct {
	CPU        ResourceProvisioning `json:"cpu"`
	Memory     ResourceProvisioning `json:"memory"`
	Efficiency float64              `json:"efficiency"`
}

const (
	OptimalUtilizationMin = 0.4
	OptimalUtilizationMax = 0.7

	CPUHeadroom    = 0.2
	MemoryHeadroom = 0.2

	// Skip over-provisioned when the request is already at the practical floor.
	MinCPURequestCores    = 0.01
	MinMemoryRequestBytes = 64 * 1024 * 1024

	ThrottlingThreshold = 0.05
	PressureThreshold   = 0.1

	BurstEffectiveMinFloor = 0.05
	BurstEffectiveMinCeil  = 0.4

	// Partial predictability credit when usage is bursty but well observed.
	confidenceBurstThreshold  = 1.6
	confidenceBurstFloor      = 0.8
	confidenceDataFactorFloor = 0.5
)

type resourceProvisioningInput struct {
	currentRequest               *float64
	currentLimit                 *float64
	percentileP95                float64
	peakUsage                    float64
	healthyRequestUtilizationMin float64
	minimumRequest               float64
	limitHeadroom                float64

	underProvisionedFromStability bool
	suppressOverProvisioned       bool
	efficiencyFromStability       float64
}

func AnalyzeProvisioning(specs ResourceSpecs, utilization UtilizationResult, stability StabilityResult, minSamples int) ProvisioningResult {
	result := ProvisioningResult{
		CPU:    analyzeCPUProvisioning(specs, utilization.CPU, stability, minSamples),
		Memory: analyzeMemoryProvisioning(specs, utilization.Memory, stability, minSamples),
	}

	const cpuWeight = 0.3
	const memoryWeight = 0.7
	result.Efficiency = (result.CPU.Efficiency * cpuWeight) + (result.Memory.Efficiency * memoryWeight)

	return result
}

func analyzeCPUProvisioning(specs ResourceSpecs, utilization ResourceUtilization, stability StabilityResult, minSamples int) ResourceProvisioning {
	healthyRequestUtilizationMin := effectiveMinFromBurstiness(
		utilization.Stats.Mean,
		utilization.Stats.Percentile.P95,
		utilization.Stats.Max,
	)
	suppressOverProvisioned := stability.CPUThrottling > 0 && stability.CPUThrottling <= ThrottlingThreshold

	underProvisionedFromStability := false
	efficiencyFromStability := 1.0
	if stability.CPUThrottling > ThrottlingThreshold {
		underProvisionedFromStability = true
		penalty := (stability.CPUThrottling - ThrottlingThreshold) * 3.0
		efficiencyFromStability = min(efficiencyFromStability, max(0.0, 1.0-penalty))
	}
	if stability.CPUPressure > PressureThreshold {
		underProvisionedFromStability = true
		penalty := (stability.CPUPressure - PressureThreshold) * 1.0
		efficiencyFromStability = min(efficiencyFromStability, max(0.0, 1.0-penalty))
	}

	return analyzeResourceProvisioning(resourceProvisioningInput{
		currentRequest:                specs.CPURequest,
		currentLimit:                  specs.CPULimit,
		percentileP95:                 utilization.Stats.Percentile.P95,
		peakUsage:                     utilization.Stats.Max,
		healthyRequestUtilizationMin:  healthyRequestUtilizationMin,
		minimumRequest:                MinCPURequestCores,
		limitHeadroom:                 CPUHeadroom,
		underProvisionedFromStability: underProvisionedFromStability,
		suppressOverProvisioned:       suppressOverProvisioned,
		efficiencyFromStability:       efficiencyFromStability,
	}, computeResourceConfidence(utilization.Stats, utilization.SampleSize, minSamples))
}

func analyzeMemoryProvisioning(specs ResourceSpecs, utilization ResourceUtilization, stability StabilityResult, minSamples int) ResourceProvisioning {
	suppressOverProvisioned := stability.MemoryPressure > 0 && stability.MemoryPressure <= PressureThreshold

	underProvisionedFromStability := false
	efficiencyFromStability := 1.0
	if stability.MemoryOOM > 0 {
		underProvisionedFromStability = true
		penalty := eventCountPenalty(stability.MemoryOOM, maxPenaltyOOM, oomCountAtMax)
		efficiencyFromStability = min(efficiencyFromStability, max(0.0, 1.0-penalty))
	}
	if stability.MemoryFailCnt > 0 {
		underProvisionedFromStability = true
		penalty := eventCountPenalty(stability.MemoryFailCnt, maxPenaltyMemoryFailCnt, memoryFailCountAtMax)
		efficiencyFromStability = min(efficiencyFromStability, max(0.0, 1.0-penalty))
	}
	if stability.MemoryPressure > PressureThreshold {
		underProvisionedFromStability = true
		penalty := (stability.MemoryPressure - PressureThreshold) * 1.0
		efficiencyFromStability = min(efficiencyFromStability, max(0.0, 1.0-penalty))
	}

	return analyzeResourceProvisioning(resourceProvisioningInput{
		currentRequest:                specs.MemoryRequest,
		currentLimit:                  specs.MemoryLimit,
		percentileP95:                 utilization.Stats.Percentile.P95,
		peakUsage:                     utilization.Stats.Max,
		healthyRequestUtilizationMin:  OptimalUtilizationMin,
		minimumRequest:                MinMemoryRequestBytes,
		limitHeadroom:                 MemoryHeadroom,
		underProvisionedFromStability: underProvisionedFromStability,
		suppressOverProvisioned:       suppressOverProvisioned,
		efficiencyFromStability:       efficiencyFromStability,
	}, computeResourceConfidence(utilization.Stats, utilization.SampleSize, minSamples))
}

func analyzeResourceProvisioning(input resourceProvisioningInput, confidence float64) ResourceProvisioning {
	result := ResourceProvisioning{
		CurrentRequest: input.currentRequest,
		CurrentLimit:   input.currentLimit,
		Efficiency:     input.efficiencyFromStability,
		Confidence:     confidence,
	}

	hasRequest := input.currentRequest != nil && *input.currentRequest > 0
	hasLimit := input.currentLimit != nil && *input.currentLimit > 0

	if !hasRequest {
		result.Status = ProvisioningUnspecified
		result.Efficiency = 0.0
		return result
	}

	underProvisioned := input.underProvisionedFromStability
	overProvisioned := false

	requestUtilization := input.percentileP95 / *input.currentRequest
	result.RequestUtilization = &requestUtilization

	requestAtMinimum := *input.currentRequest <= input.minimumRequest
	if requestUtilization > OptimalUtilizationMax {
		underProvisioned = true
	}
	if requestUtilization < input.healthyRequestUtilizationMin && !requestAtMinimum && !input.suppressOverProvisioned {
		overProvisioned = true
	}

	result.Efficiency = min(result.Efficiency, requestFitEfficiency(
		requestUtilization,
		input.healthyRequestUtilizationMin,
		requestAtMinimum,
		input.suppressOverProvisioned,
	))

	if hasLimit {
		limitUtilization := input.peakUsage / *input.currentLimit
		result.LimitUtilization = &limitUtilization

		if limitUtilization > (1.0 - input.limitHeadroom) {
			underProvisioned = true
			limitPenalty := 1.0
			if limitUtilization > 1.0 {
				limitPenalty = 0.0
			} else {
				limitPenalty = (1.0 - limitUtilization) / input.limitHeadroom
			}
			result.Efficiency = min(result.Efficiency, limitPenalty)
		}
	}

	if underProvisioned {
		result.Status = ProvisioningUnderProvisioned
	} else if overProvisioned {
		result.Status = ProvisioningOverProvisioned
	} else {
		result.Status = ProvisioningOptimal
	}

	result.Efficiency = max(0.0, min(1.0, result.Efficiency))
	return result
}

func requestFitEfficiency(
	requestUtilization float64,
	healthyRequestUtilizationMin float64,
	requestAtMinimum bool,
	suppressOverProvisioned bool,
) float64 {
	if (requestUtilization >= healthyRequestUtilizationMin && requestUtilization <= OptimalUtilizationMax) ||
		(requestUtilization < healthyRequestUtilizationMin && (requestAtMinimum || suppressOverProvisioned)) {
		return 1.0
	}
	if requestUtilization < healthyRequestUtilizationMin {
		return requestUtilization / healthyRequestUtilizationMin
	}
	if requestUtilization > 1.0 {
		return 0.0
	}
	return 1.0 - ((requestUtilization - OptimalUtilizationMax) / (1.0 - OptimalUtilizationMax))
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

func burstScore(mean, percentileP95, peakUsage float64) float64 {
	if mean <= 0 {
		return 0
	}
	return (percentileP95/mean + peakUsage/mean) / 2.0
}

// effectiveMinFromBurstiness lowers the minimum healthy request utilization when mean, P95, and peak
// show burstiness, so bursty CPU workloads are not marked over-provisioned for sitting near idle between spikes.
func effectiveMinFromBurstiness(mean, percentileP95, peakUsage float64) float64 {
	if mean <= 0 {
		return BurstEffectiveMinCeil
	}
	score := max(burstScore(mean, percentileP95, peakUsage), 1.0)
	effectiveMin := BurstEffectiveMinCeil - (score-1.0)*0.1
	effectiveMin = max(effectiveMin, BurstEffectiveMinFloor)
	effectiveMin = min(effectiveMin, BurstEffectiveMinCeil)
	return effectiveMin
}
