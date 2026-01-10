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
	OptimalUtilizationMin = 0.50 // 50%
	OptimalUtilizationMax = 0.70 // 70%
	// Headroom for limits (peak should stay below this percentage of limit)
	LimitHeadroom = 0.2 // 20% headroom
)

// Analyzes CPU provisioning based on specs and utilization
func AnalyzeCPUProvisioning(specs ResourceSpecs, utilization CPUUtilization) (CPUProvisioning, error) {
	result := CPUProvisioning{
		IsOverProvisioned:  false,
		IsUnderProvisioned: false,
		Efficiency:         0.5, // Default neutral
		Confidence:         0.0,
	}

	// Calculate confidence based on data quality
	// More data points and lower variance = higher confidence
	if utilization.Stats.Mean == 0 {
		// If mean is zero, we can't calculate confidence reliably
		result.Confidence = 0.0
	} else if utilization.Stats.StdDev > 0 {
		// Simplified confidence: based on coefficient of variation
		cv := utilization.Stats.StdDev / utilization.Stats.Mean
		result.Confidence = math.Min(1.0, 1.0/(1.0+cv))
	} else {
		// Zero variance (all values are the same)
		result.Confidence = 1.0
	}

	// Analyze request utilization
	if specs.CPURequest != nil && *specs.CPURequest > 0 {
		// Use P95 for request utilization
		result.RequestUtilization = utilization.Stats.Percentile.P95 / *specs.CPURequest

		// Check for over-provisioning (P95 usage below optimal range)
		if result.RequestUtilization < OptimalUtilizationMin {
			result.IsOverProvisioned = true
		}

		// Check for under-provisioning on requests (P95 exceeds optimal range)
		if result.RequestUtilization > OptimalUtilizationMax {
			result.IsUnderProvisioned = true
		}
	}

	// Analyze limit utilization
	if specs.CPULimit != nil && *specs.CPULimit > 0 {
		result.LimitUtilization = utilization.Stats.Max / *specs.CPULimit

		// Check for under-provisioning on limits (peak too close to limit, needs headroom)
		if result.LimitUtilization > (1.0 - LimitHeadroom) {
			result.IsUnderProvisioned = true
		}
	}

	// Calculate efficiency score
	// Efficiency is higher when utilization is in optimal range
	if specs.CPURequest != nil && *specs.CPURequest > 0 {
		utilRatio := result.RequestUtilization
		if utilRatio >= OptimalUtilizationMin && utilRatio <= OptimalUtilizationMax {
			result.Efficiency = 1.0 // Optimal
		} else if utilRatio < OptimalUtilizationMin {
			// Over-provisioned: efficiency decreases as ratio decreases
			result.Efficiency = utilRatio / OptimalUtilizationMin
		} else {
			// Under-provisioned or approaching limit
			if utilRatio > 1.0 {
				result.Efficiency = 0.0 // Exceeding request
			} else {
				// Between optimal max and 1.0
				result.Efficiency = 1.0 - ((utilRatio - OptimalUtilizationMax) / (1.0 - OptimalUtilizationMax))
			}
		}
		// Clamp to 0-1
		result.Efficiency = math.Max(0.0, math.Min(1.0, result.Efficiency))
	}

	return result, nil
}

// Analyzes memory provisioning based on specs and utilization
func AnalyzeMemoryProvisioning(specs ResourceSpecs, utilization MemoryUtilization) (MemoryProvisioning, error) {
	result := MemoryProvisioning{
		IsOverProvisioned:  false,
		IsUnderProvisioned: false,
		Efficiency:         0.5, // Default neutral
		Confidence:         0.0,
	}

	// Calculate confidence based on data quality
	if utilization.Stats.Mean == 0 {
		// If mean is zero, we can't calculate confidence reliably
		result.Confidence = 0.0
	} else if utilization.Stats.StdDev > 0 {
		// Simplified confidence: based on coefficient of variation
		cv := utilization.Stats.StdDev / utilization.Stats.Mean
		result.Confidence = math.Min(1.0, 1.0/(1.0+cv))
	} else {
		// Zero variance (all values are the same)
		result.Confidence = 1.0
	}

	// Analyze request utilization
	if specs.MemoryRequest != nil && *specs.MemoryRequest > 0 {
		// Use P95 for request utilization
		result.RequestUtilization = utilization.Stats.Percentile.P95 / *specs.MemoryRequest

		// Check for over-provisioning (P95 usage below optimal range)
		if result.RequestUtilization < OptimalUtilizationMin {
			result.IsOverProvisioned = true
		}

		// Check for under-provisioning on requests (P95 exceeds optimal range)
		if result.RequestUtilization > OptimalUtilizationMax {
			result.IsUnderProvisioned = true
		}
	}

	// Analyze limit utilization
	if specs.MemoryLimit != nil && *specs.MemoryLimit > 0 {
		result.LimitUtilization = utilization.Stats.Max / *specs.MemoryLimit

		// Check for under-provisioning on limits (peak too close to limit, needs headroom)
		if result.LimitUtilization > (1.0 - LimitHeadroom) {
			result.IsUnderProvisioned = true
		}
	}

	// Calculate efficiency score
	if specs.MemoryRequest != nil && *specs.MemoryRequest > 0 {
		utilRatio := result.RequestUtilization
		if utilRatio >= OptimalUtilizationMin && utilRatio <= OptimalUtilizationMax {
			result.Efficiency = 1.0 // Optimal
		} else if utilRatio < OptimalUtilizationMin {
			// Over-provisioned: efficiency decreases as ratio decreases
			result.Efficiency = utilRatio / OptimalUtilizationMin
		} else {
			// Under-provisioned or approaching limit
			if utilRatio > 1.0 {
				result.Efficiency = 0.0 // Exceeding request
			} else {
				// Between optimal max and 1.0
				result.Efficiency = 1.0 - ((utilRatio - OptimalUtilizationMax) / (1.0 - OptimalUtilizationMax))
			}
		}
		// Clamp to 0-1
		result.Efficiency = math.Max(0.0, math.Min(1.0, result.Efficiency))
	}

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
	cpuWeight := 0.5
	memoryWeight := 0.5
	result.Efficiency = (result.CPU.Efficiency * cpuWeight) + (result.Memory.Efficiency * memoryWeight)

	return result, nil
}
