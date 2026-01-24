package compute

import (
	"fmt"
	"math"
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
	RequestUtilization float64 `json:"request_utilization"` // usage / request (0-1+)
	LimitUtilization   float64 `json:"limit_utilization"`   // usage / limit (0-1+)
	IsOverProvisioned  bool    `json:"is_over_provisioned"`
	IsUnderProvisioned bool    `json:"is_under_provisioned"`
	Efficiency         float64 `json:"efficiency"` // 0-1 score (higher is better)
	Confidence         float64 `json:"confidence"` // 0-1 score based on data quality
}

// Represents memory provisioning analysis results
type MemoryProvisioning struct {
	RequestUtilization float64 `json:"request_utilization"` // usage / request (0-1+)
	LimitUtilization   float64 `json:"limit_utilization"`   // usage / limit (0-1+)
	IsOverProvisioned  bool    `json:"is_over_provisioned"`
	IsUnderProvisioned bool    `json:"is_under_provisioned"`
	Efficiency         float64 `json:"efficiency"` // 0-1 score (higher is better)
	Confidence         float64 `json:"confidence"` // 0-1 score based on data quality
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

	// Burst factor for CPU (if limit/request ratio is high)
	BurstFactorThreshold = 5.0
	BurstOptimalMin      = 0.1 // Allow lower request utilization for bursty workloads
)

// Analyzes CPU provisioning based on specs and utilization
func AnalyzeCPUProvisioning(specs ResourceSpecs, utilization CPUUtilization) (CPUProvisioning, error) {
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
		// Confidence: based on coefficient of variation
		cv := utilization.Stats.StdDev / utilization.Stats.Mean
		result.Confidence = math.Min(1.0, 1.0/(1.0+cv))
	} else {
		// Zero variance (all values are the same)
		result.Confidence = 1.0
	}

	hasRequest := specs.CPURequest != nil && *specs.CPURequest > 0
	hasLimit := specs.CPULimit != nil && *specs.CPULimit > 0

	// Handle missing resources
	if !hasRequest {
		result.IsUnderProvisioned = true
		result.Efficiency = math.Min(result.Efficiency, 0.2)
	}
	if !hasLimit {
		result.IsUnderProvisioned = true
		result.Efficiency = math.Min(result.Efficiency, 0.2)
	}

	// Analyze request utilization
	if hasRequest {
		result.RequestUtilization = utilization.Stats.Percentile.P95 / *specs.CPURequest

		minThreshold := OptimalUtilizationMin
		// Handle burst factor: if limit is much higher than request, more lenient
		if hasLimit && (*specs.CPULimit / *specs.CPURequest) >= BurstFactorThreshold {
			minThreshold = BurstOptimalMin
		}

		// Check for over-provisioning (P95 usage below optimal range)
		if result.RequestUtilization < minThreshold {
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
		} else if result.RequestUtilization < minThreshold {
			requestEfficiency = result.RequestUtilization / minThreshold
		} else {
			if result.RequestUtilization > 1.0 {
				requestEfficiency = 0.0
			} else {
				requestEfficiency = 1.0 - ((result.RequestUtilization - OptimalUtilizationMax) / (1.0 - OptimalUtilizationMax))
			}
		}
		result.Efficiency = math.Min(result.Efficiency, requestEfficiency)
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
			result.Efficiency = math.Min(result.Efficiency, limitPenalty)
		}
	}

	// Clamp to 0-1
	result.Efficiency = math.Max(0.0, math.Min(1.0, result.Efficiency))

	return result, nil
}

// Analyzes memory provisioning based on specs and utilization
func AnalyzeMemoryProvisioning(specs ResourceSpecs, utilization MemoryUtilization) (MemoryProvisioning, error) {
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
		// Confidence: based on coefficient of variation
		cv := utilization.Stats.StdDev / utilization.Stats.Mean
		result.Confidence = math.Min(1.0, 1.0/(1.0+cv))
	} else {
		// Zero variance (all values are the same)
		result.Confidence = 1.0
	}

	hasRequest := specs.MemoryRequest != nil && *specs.MemoryRequest > 0
	hasLimit := specs.MemoryLimit != nil && *specs.MemoryLimit > 0

	// Handle missing resources
	if !hasRequest {
		result.IsUnderProvisioned = true
		result.Efficiency = math.Min(result.Efficiency, 0.2)
	}
	if !hasLimit {
		result.IsUnderProvisioned = true
		result.Efficiency = math.Min(result.Efficiency, 0.1)
	}

	// Analyze request utilization
	if hasRequest {
		result.RequestUtilization = utilization.Stats.Percentile.P95 / *specs.MemoryRequest

		// Check for over-provisioning (P95 usage below optimal range)
		if result.RequestUtilization < OptimalUtilizationMin {
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
		} else if result.RequestUtilization < OptimalUtilizationMin {
			requestEfficiency = result.RequestUtilization / OptimalUtilizationMin
		} else {
			if result.RequestUtilization > 1.0 {
				requestEfficiency = 0.0
			} else {
				requestEfficiency = 1.0 - ((result.RequestUtilization - OptimalUtilizationMax) / (1.0 - OptimalUtilizationMax))
			}
		}
		result.Efficiency = math.Min(result.Efficiency, requestEfficiency)
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
			result.Efficiency = math.Min(result.Efficiency, limitPenalty)
		}
	}

	// Clamp to 0-1
	result.Efficiency = math.Max(0.0, math.Min(1.0, result.Efficiency))

	return result, nil
}

// Analyzes resource provisioning from specs and utilization
func AnalyzeProvisioning(specs ResourceSpecs, utilization UtilizationResult) (ProvisioningResult, error) {
	var result ProvisioningResult
	var err error

	// Analyze CPU provisioning
	result.CPU, err = AnalyzeCPUProvisioning(specs, utilization.CPU)
	if err != nil {
		return ProvisioningResult{}, fmt.Errorf("failed to analyze CPU provisioning: %w", err)
	}

	// Analyze memory provisioning
	result.Memory, err = AnalyzeMemoryProvisioning(specs, utilization.Memory)
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
