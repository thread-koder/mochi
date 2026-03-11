package compute

import (
	"fmt"
)

// Represents resource specifications (requests and limits)
type ResourceSpecs struct {
	CPURequest    *float64 `json:"cpu_request"`    // CPU cores (e.g., 0.1 for "100m")
	CPULimit      *float64 `json:"cpu_limit"`      // CPU cores
	MemoryRequest *float64 `json:"memory_request"` // Memory in bytes
	MemoryLimit   *float64 `json:"memory_limit"`   // Memory in bytes
}

// Represents CPU provisioning analysis results
type CPUProvisioning struct {
	RequestUtilization float64  `json:"request_utilization"` // usage / request (0-1+)
	LimitUtilization   float64  `json:"limit_utilization"`   // usage / limit (0-1+)
	CurrentRequest     *float64 `json:"current_request,omitempty"`
	CurrentLimit       *float64 `json:"current_limit,omitempty"`
	IsOverProvisioned  bool     `json:"is_over_provisioned"`
	IsUnderProvisioned bool     `json:"is_under_provisioned"`
	Efficiency         float64  `json:"efficiency"` // 0-1 score (higher is better)
	Confidence         float64  `json:"confidence"` // 0-1 score based on data quality
}

// Represents memory provisioning analysis results
type MemoryProvisioning struct {
	RequestUtilization float64  `json:"request_utilization"` // usage / request (0-1+)
	LimitUtilization   float64  `json:"limit_utilization"`   // usage / limit (0-1+)
	CurrentRequest     *float64 `json:"current_request,omitempty"`
	CurrentLimit       *float64 `json:"current_limit,omitempty"`
	IsOverProvisioned  bool     `json:"is_over_provisioned"`
	IsUnderProvisioned bool     `json:"is_under_provisioned"`
	Efficiency         float64  `json:"efficiency"` // 0-1 score (higher is better)
	Confidence         float64  `json:"confidence"` // 0-1 score based on data quality
}

// Represents overall provisioning analysis results
type ProvisioningResult struct {
	CPU        CPUProvisioning    `json:"cpu"`
	Memory     MemoryProvisioning `json:"memory"`
	Efficiency float64            `json:"efficiency"` // Overall efficiency score (0-1)
}

// Thresholds for provisioning detection
const (
	// Optimal utilization range for requests
	OptimalUtilizationMin = 0.4 // 40%
	OptimalUtilizationMax = 0.7 // 70%

	// Headroom for limits (peak should stay below this percentage of limit)
	CPUHeadroom    = 0.2 // 20% headroom (80% utilization)
	MemoryHeadroom = 0.2 // 20% headroom (80% utilization)

	// Minimum request floors (at or below these cannot be over-provisioned)
	MinCPURequestCores    = 0.01             // 10m
	MinMemoryRequestBytes = 64 * 1024 * 1024 // 64Mi

	// Stability context thresholds
	ThrottlingThreshold = 0.05 // 5% (throttling above this is under-provisioned)
	PressureThreshold   = 0.1  // 10% (pressure above this is under-provisioned)

	// Bounds for dynamic minimum request utilization
	BurstEffectiveMinFloor = 0.05 // Lower bound for dynamic min (5%)
	BurstEffectiveMinCeil  = 0.4  // Upper bound for dynamic min (40%)
)

// Analyzes CPU provisioning based on specs, utilization, and stability
func AnalyzeCPUProvisioning(specs ResourceSpecs, utilization CPUUtilization, stability StabilityResult) (CPUProvisioning, error) {
	result := CPUProvisioning{
		IsOverProvisioned:  false,
		IsUnderProvisioned: false,
		Efficiency:         1.0, // Start at optimal then penalize
		Confidence:         0.0,
	}

	// Calculate confidence
	// More data points and lower variance = higher confidence
	if utilization.Stats.Mean == 0 {
		// If mean is zero, we can't calculate confidence reliably
		result.Confidence = 0.0
	} else if utilization.Stats.StdDev > 0 {
		// Calculate confidence based on coefficient of variation
		cv := utilization.Stats.StdDev / utilization.Stats.Mean
		result.Confidence = min(1.0, 1.0/(1.0+cv))
	} else {
		// Zero variance (all values are the same)
		result.Confidence = 1.0
	}

	hasRequest := specs.CPURequest != nil && *specs.CPURequest > 0
	hasLimit := specs.CPULimit != nil && *specs.CPULimit > 0

	result.CurrentRequest = specs.CPURequest
	result.CurrentLimit = specs.CPULimit

	// If either request or limit is missing, treat as under-provisioned
	if !hasRequest || !hasLimit {
		result.IsUnderProvisioned = true
	}

	// If both request and limit are missing, treat as completely inefficient
	if !hasRequest && !hasLimit {
		result.Efficiency = 0.0
		return result, nil
	}

	// Force under-provisioned when throttling or pressure metrics are above thresholds
	if stability.CPUThrottling > ThrottlingThreshold {
		result.IsUnderProvisioned = true
		penalty := (stability.CPUThrottling - ThrottlingThreshold) * 2.0
		result.Efficiency = min(result.Efficiency, max(0.0, 1.0-penalty))
	}
	if stability.CPUPressure > PressureThreshold {
		result.IsUnderProvisioned = true
		penalty := (stability.CPUPressure - PressureThreshold) * 0.5
		result.Efficiency = min(result.Efficiency, max(0.0, 1.0-penalty))
	}

	// Analyze request utilization
	if hasRequest {
		result.RequestUtilization = utilization.Stats.Percentile.P95 / *specs.CPURequest

		minThreshold := effectiveMinFromBurstiness(
			utilization.Stats.Mean,
			utilization.Stats.Percentile.P95,
			utilization.Stats.Max,
		)
		atFloor := *specs.CPURequest <= MinCPURequestCores
		lowThrottling := stability.CPUThrottling > 0 && stability.CPUThrottling <= ThrottlingThreshold

		// Check for over-provisioning (P95 usage below optimal range)
		// Treat as optimal when at floor or low throttling
		if result.RequestUtilization < minThreshold && !atFloor && !lowThrottling {
			result.IsOverProvisioned = true
		}

		// Check for under-provisioning on requests (P95 exceeds optimal range)
		if result.RequestUtilization > OptimalUtilizationMax {
			result.IsUnderProvisioned = true
		}

		// Calculate request-based efficiency
		var requestEfficiency float64
		if result.RequestUtilization >= minThreshold && result.RequestUtilization <= OptimalUtilizationMax {
			requestEfficiency = 1.0
		} else if atFloor || lowThrottling {
			// Treat as optimal despite low utilization
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

	// Analyze limit utilization
	if hasLimit {
		result.LimitUtilization = utilization.Stats.Max / *specs.CPULimit

		// Check for under-provisioning on limits (peak too close to limit, needs headroom)
		if result.LimitUtilization > (1.0 - CPUHeadroom) {
			result.IsUnderProvisioned = true
			// Penalize efficiency for approaching limits
			limitPenalty := 1.0
			if result.LimitUtilization > 1.0 {
				limitPenalty = 0.0
			} else {
				limitPenalty = (1.0 - result.LimitUtilization) / CPUHeadroom
			}
			result.Efficiency = min(result.Efficiency, limitPenalty)
		}
	}

	// If exactly one of request or limit is missing, treat as partially inefficient
	if (hasRequest && !hasLimit) || (!hasRequest && hasLimit) {
		result.Efficiency = min(result.Efficiency, 0.5)
	}

	// Clamp efficiency to 0-1
	result.Efficiency = max(0.0, min(1.0, result.Efficiency))

	return result, nil
}

// Analyzes memory provisioning based on specs, utilization, and stability
func AnalyzeMemoryProvisioning(specs ResourceSpecs, utilization MemoryUtilization, stability StabilityResult) (MemoryProvisioning, error) {
	result := MemoryProvisioning{
		IsOverProvisioned:  false,
		IsUnderProvisioned: false,
		Efficiency:         1.0, // Start at optimal then penalize
		Confidence:         0.0,
	}

	// Calculate confidence based on data quality
	if utilization.Stats.Mean == 0 {
		// If mean is zero, we can't calculate confidence reliably
		result.Confidence = 0.0
	} else if utilization.Stats.StdDev > 0 {
		// Calculate confidence based on coefficient of variation
		cv := utilization.Stats.StdDev / utilization.Stats.Mean
		result.Confidence = min(1.0, 1.0/(1.0+cv))
	} else {
		// Zero variance (all values are the same)
		result.Confidence = 1.0
	}

	hasRequest := specs.MemoryRequest != nil && *specs.MemoryRequest > 0
	hasLimit := specs.MemoryLimit != nil && *specs.MemoryLimit > 0

	result.CurrentRequest = specs.MemoryRequest
	result.CurrentLimit = specs.MemoryLimit

	// If either request or limit is missing, treat as under-provisioned
	if !hasRequest || !hasLimit {
		result.IsUnderProvisioned = true
	}

	// If both request and limit are missing, treat as completely inefficient
	if !hasRequest && !hasLimit {
		result.Efficiency = 0.0
		return result, nil
	}

	// Force under-provisioned when OOM, failcnt, or pressure are above threshold
	if stability.MemoryOOM > 0 {
		result.IsUnderProvisioned = true
		penalty := min(stability.MemoryOOM*0.5, 0.35)
		result.Efficiency = min(result.Efficiency, max(0.0, 1.0-penalty))
	}
	if stability.MemoryFailCnt > 0 {
		result.IsUnderProvisioned = true
		penalty := min(stability.MemoryFailCnt*0.2, 0.25)
		result.Efficiency = min(result.Efficiency, max(0.0, 1.0-penalty))
	}
	if stability.MemoryPressure > PressureThreshold {
		result.IsUnderProvisioned = true
		penalty := (stability.MemoryPressure - PressureThreshold) * 1.0
		result.Efficiency = min(result.Efficiency, max(0.0, 1.0-penalty))
	}

	// Analyze request utilization
	if hasRequest {
		result.RequestUtilization = utilization.Stats.Percentile.P95 / *specs.MemoryRequest

		atFloor := *specs.MemoryRequest <= MinMemoryRequestBytes

		// Check for over-provisioning (P95 usage below optimal range)
		// Treat as optimal when at floor
		if result.RequestUtilization < OptimalUtilizationMin && !atFloor {
			result.IsOverProvisioned = true
		}

		// Check for under-provisioning on requests (P95 exceeds optimal range)
		if result.RequestUtilization > OptimalUtilizationMax {
			result.IsUnderProvisioned = true
		}

		// Calculate request-based efficiency
		var requestEfficiency float64
		if result.RequestUtilization >= OptimalUtilizationMin && result.RequestUtilization <= OptimalUtilizationMax {
			requestEfficiency = 1.0
		} else if atFloor {
			// Treat as optimal despite low utilization
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

	// Analyze limit utilization
	if hasLimit {
		result.LimitUtilization = utilization.Stats.Max / *specs.MemoryLimit

		// Check for under-provisioning on limits (peak too close to limit, needs headroom)
		if result.LimitUtilization > (1.0 - MemoryHeadroom) {
			result.IsUnderProvisioned = true
			// Penalize efficiency for approaching limits
			limitPenalty := 1.0
			if result.LimitUtilization > 1.0 {
				limitPenalty = 0.0
			} else {
				limitPenalty = (1.0 - result.LimitUtilization) / MemoryHeadroom
			}
			result.Efficiency = min(result.Efficiency, limitPenalty)
		}
	}

	// If exactly one of request or limit is missing, treat as partially inefficient
	if (hasRequest && !hasLimit) || (!hasRequest && hasLimit) {
		result.Efficiency = min(result.Efficiency, 0.5)
	}

	// Clamp efficiency to 0-1
	result.Efficiency = max(0.0, min(1.0, result.Efficiency))

	return result, nil
}

// Analyzes resource provisioning from specs, utilization, and stability
func AnalyzeProvisioning(specs ResourceSpecs, utilization UtilizationResult, stability StabilityResult) (ProvisioningResult, error) {
	var result ProvisioningResult
	var err error

	// Analyze CPU provisioning
	result.CPU, err = AnalyzeCPUProvisioning(specs, utilization.CPU, stability)
	if err != nil {
		return ProvisioningResult{}, fmt.Errorf("failed to analyze CPU provisioning: %w", err)
	}

	// Analyze memory provisioning
	result.Memory, err = AnalyzeMemoryProvisioning(specs, utilization.Memory, stability)
	if err != nil {
		return ProvisioningResult{}, fmt.Errorf("failed to analyze memory provisioning: %w", err)
	}

	// Calculate overall efficiency (weighted average)
	// Memory is more critical
	cpuWeight := 0.3
	memoryWeight := 0.7
	result.Efficiency = (result.CPU.Efficiency * cpuWeight) + (result.Memory.Efficiency * memoryWeight)

	return result, nil
}

// Returns dynamic minimum request utilization from usage pattern
func effectiveMinFromBurstiness(mean, p95, peak float64) float64 {
	if mean <= 0 {
		return BurstEffectiveMinCeil
	}
	// Burstiness score from P95/mean and peak/mean (higher ratio = more bursty)
	burstScore := (p95/mean + peak/mean) / 2.0
	// Clamp burst score to 1.0
	burstScore = max(burstScore, 1.0)
	// Effective min decreases as burstiness increases (allow lower util for bursty workloads)
	effectiveMin := BurstEffectiveMinCeil - (burstScore-1.0)*0.1
	// Clamp effective min to the floor and ceil
	effectiveMin = max(effectiveMin, BurstEffectiveMinFloor)
	effectiveMin = min(effectiveMin, BurstEffectiveMinCeil)
	return effectiveMin
}
