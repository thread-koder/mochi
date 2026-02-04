package compute

import (
	"fmt"
	"math"
)

// Represents recommended resource values
type ResourceRecommendation struct {
	CPURequest    *string `json:"cpu_request"`    // Recommended CPU request (e.g., "100m", "0.5")
	CPULimit      *string `json:"cpu_limit"`      // Recommended CPU limit
	MemoryRequest *string `json:"memory_request"` // Recommended memory request (e.g., "128Mi", "1Gi")
	MemoryLimit   *string `json:"memory_limit"`   // Recommended memory limit
	Confidence    float64 `json:"confidence"`     // Overall confidence score (0-1)
}

// Represents the recommendation mode
type RecommendationMode string

const (
	ModeCostOptimized RecommendationMode = "cost_optimized" // Maximum cost savings, accept throttling risk
	ModeBurstable     RecommendationMode = "burstable"      // Default: balance performance/reliability/efficiency
	ModeGuaranteed    RecommendationMode = "guaranteed"     // Best performance, no throttling risk
)

// Configuration for recommendation calculations
type RecommendationConfig struct {
	// Recommendation mode: cost_optimized (maximize cost savings), burstable (balance), or guaranteed (best performance)
	Mode RecommendationMode
	// Safety margin multiplier for requests (default: 1.2 = 20% headroom)
	RequestSafetyMargin float64
	// Safety margin multiplier for limits based on peak (default: 1.3 = 30% headroom)
	LimitSafetyMargin float64
	// Base margin for cost-optimized mode requests (default: 1.15 = 15% headroom)
	CostOptimizedRequestMargin float64
	// Base margin for cost-optimized mode limits (default: 1.2 = 20% headroom)
	CostOptimizedLimitMargin float64
	// Multiplier for guaranteed mode margins (default: 1.1)
	GuaranteedMarginMultiplier float64
	// Minimum CPU request in cores (default: 0.01 = 10m)
	MinCPURequest float64
	// Minimum memory request in bytes (default: 64Mi)
	MinMemoryRequest int64
	// Minimum confidence threshold to generate recommendations (default: 0.5)
	MinConfidenceThreshold float64
	// Burst detection threshold: if Max > percentile * BurstThreshold, treat as burst workload
	BurstThreshold float64
}

// Returns default recommendation configuration
func DefaultRecommendationConfig() RecommendationConfig {
	return RecommendationConfig{
		Mode:                       ModeBurstable,    // Default: optimize for efficiency
		RequestSafetyMargin:        1.2,              // 20% headroom
		LimitSafetyMargin:          1.3,              // 30% headroom
		CostOptimizedRequestMargin: 1.15,             // 15% headroom for cost-optimized requests
		CostOptimizedLimitMargin:   1.2,              // 20% headroom for cost-optimized limits
		GuaranteedMarginMultiplier: 1.1,              // 1.1x multiplier for guaranteed mode margins
		MinCPURequest:              0.01,             // 10m minimum
		MinMemoryRequest:           64 * 1024 * 1024, // 64Mi minimum
		MinConfidenceThreshold:     0.5,              // 50% minimum confidence
		BurstThreshold:             1.8,              // Max > 1.8x percentile = burst workload
	}
}

// Validates recommendation configuration
func (config RecommendationConfig) Validate() error {
	if config.Mode != ModeCostOptimized && config.Mode != ModeBurstable && config.Mode != ModeGuaranteed {
		return fmt.Errorf("Mode must be one of %s, %s, or %s, got: %v", ModeCostOptimized, ModeBurstable, ModeGuaranteed, config.Mode)
	}
	if config.RequestSafetyMargin <= 0 {
		return fmt.Errorf("RequestSafetyMargin must be positive, got: %v", config.RequestSafetyMargin)
	}
	if config.LimitSafetyMargin <= 0 {
		return fmt.Errorf("LimitSafetyMargin must be positive, got: %v", config.LimitSafetyMargin)
	}
	if config.CostOptimizedRequestMargin <= 0 {
		return fmt.Errorf("CostOptimizedRequestMargin must be positive, got: %v", config.CostOptimizedRequestMargin)
	}
	if config.CostOptimizedLimitMargin <= 0 {
		return fmt.Errorf("CostOptimizedLimitMargin must be positive, got: %v", config.CostOptimizedLimitMargin)
	}
	if config.GuaranteedMarginMultiplier <= 0 {
		return fmt.Errorf("GuaranteedMarginMultiplier must be positive, got: %v", config.GuaranteedMarginMultiplier)
	}
	if config.MinCPURequest < 0 {
		return fmt.Errorf("MinCPURequest must be non-negative, got: %v", config.MinCPURequest)
	}
	if config.MinMemoryRequest < 0 {
		return fmt.Errorf("MinMemoryRequest must be non-negative, got: %v", config.MinMemoryRequest)
	}
	if config.MinConfidenceThreshold < 0 || config.MinConfidenceThreshold > 1 {
		return fmt.Errorf("MinConfidenceThreshold must be between 0 and 1, got: %v", config.MinConfidenceThreshold)
	}
	if config.BurstThreshold <= 0 {
		return fmt.Errorf("BurstThreshold must be positive, got: %v", config.BurstThreshold)
	}
	return nil
}

// Calculates CPU request recommendation based on utilization analysis
func CalculateCPURequestRecommendation(
	currentRequest *float64,
	utilization CPUUtilization,
	provisioning CPUProvisioning,
	stability StabilityResult,
	config RecommendationConfig,
) (*float64, error) {
	// Adjust confidence threshold based on efficiency
	effectiveThreshold := config.MinConfidenceThreshold
	if provisioning.Efficiency < 0.3 {
		// Low efficiency = be more aggressive (lower threshold)
		effectiveThreshold = effectiveThreshold * 0.8
	}

	// If we don't have enough confidence, don't recommend
	if provisioning.Confidence < effectiveThreshold {
		return nil, nil
	}

	// Start with P95 as baseline (steady-state usage)
	percentileP95 := utilization.Stats.Percentile.P95
	percentileP99 := utilization.Stats.Percentile.P99
	peakUsage := utilization.Stats.Max

	// Calculate coefficient of variation
	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	// Detect bursty workloads and adjust percentile
	adjustedPercentile, isBursty := detectAndAdjustBurstyWorkload(
		percentileP95, percentileP99, peakUsage, config.BurstThreshold,
	)

	// Calculate pressure factors
	cpuThrottlingFactor := calculateCPUThrottlingPressureFactor(stability.CPUThrottling)
	cpuPressureFactor := calculateCPUPressureFactor(stability.CPUPressure)
	pressureFactor := cpuThrottlingFactor * cpuPressureFactor

	// Calculate recommended request based on mode
	var recommendedCores float64
	switch config.Mode {
	case ModeCostOptimized:
		// Cost-optimized mode: use adjusted percentile
		baseMargin := config.CostOptimizedRequestMargin
		safetyMargin := calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			isBursty,
		)
		recommendedCores = adjustedPercentile * safetyMargin * pressureFactor
	case ModeGuaranteed:
		// Guaranteed mode: use peak usage
		baseMargin := config.RequestSafetyMargin * config.GuaranteedMarginMultiplier
		safetyMargin := calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			isBursty,
		)
		recommendedCores = peakUsage * safetyMargin * pressureFactor
	default:
		// Burstable mode: use adjusted percentile
		baseMargin := config.RequestSafetyMargin // Standard base
		safetyMargin := calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			isBursty,
		)
		recommendedCores = adjustedPercentile * safetyMargin * pressureFactor
	}

	// Validation: ensure recommended accounts for actual usage patterns
	// Only use Mean if it's higher than the recommendation
	if utilization.Stats.Mean > 0 && recommendedCores < utilization.Stats.Mean {
		meanBased := utilization.Stats.Mean * 1.15
		if meanBased > recommendedCores {
			recommendedCores = meanBased
		}
	}

	// Apply minimum
	if recommendedCores < config.MinCPURequest {
		recommendedCores = config.MinCPURequest
	}

	// Round to reasonable precision (3 decimal places for CPU)
	recommendedCores = math.Round(recommendedCores*1000) / 1000

	// Allow larger changes for severely mis-provisioned workloads
	if currentRequest != nil {
		current := *currentRequest
		// If current is 0, always recommend (no request set)
		if current > 0 {
			diff := math.Abs(recommendedCores - current)
			changePercent := diff / current

			// For severely mis-provisioned workloads, use a lower threshold
			threshold := 0.1
			if changePercent > 1.0 {
				// More than 100% difference - severely mis-provisioned
				threshold = 0.05
			}

			if changePercent < threshold {
				// Change is too small, consider it optimal
				return nil, nil
			}
		}
	}

	return &recommendedCores, nil
}

// Calculates CPU limit recommendation based on utilization analysis
func CalculateCPULimitRecommendation(
	currentLimit *float64,
	utilization CPUUtilization,
	provisioning CPUProvisioning,
	stability StabilityResult,
	config RecommendationConfig,
	recommendedRequest *float64,
) (*float64, error) {
	// Adjust confidence threshold based on efficiency
	effectiveThreshold := config.MinConfidenceThreshold
	if provisioning.Efficiency < 0.3 {
		effectiveThreshold = effectiveThreshold * 0.8
	}

	// If we don't have enough confidence, don't recommend
	if provisioning.Confidence < effectiveThreshold {
		return nil, nil
	}

	// Use peak (max) for limit recommendation
	peakUsage := utilization.Stats.Max

	// Calculate coefficient of variation
	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	// Calculate pressure factors
	cpuThrottlingFactor := calculateCPUThrottlingPressureFactor(stability.CPUThrottling)
	cpuPressureFactor := calculateCPUPressureFactor(stability.CPUPressure)
	pressureFactor := cpuThrottlingFactor * cpuPressureFactor

	// Calculate safety margin based on mode
	var safetyMargin float64
	switch config.Mode {
	case ModeCostOptimized:
		// Cost-optimized mode: minimal base margin
		baseMargin := config.CostOptimizedLimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false, // Limits don't need burst detection
		)
	case ModeGuaranteed:
		// Guaranteed mode: higher base margin
		baseMargin := config.LimitSafetyMargin * config.GuaranteedMarginMultiplier
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false, // Limits don't need burst detection
		)
	default:
		// Burstable mode: base margin
		baseMargin := config.LimitSafetyMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false, // Limits don't need burst detection
		)
	}

	// Calculate recommended limit
	recommendedCores := peakUsage * safetyMargin * pressureFactor

	// Apply minimum
	if recommendedCores < config.MinCPURequest {
		recommendedCores = config.MinCPURequest
	}

	// Ensure limit is at least equal to recommended request
	if recommendedRequest != nil && recommendedCores < *recommendedRequest {
		recommendedCores = *recommendedRequest
	}

	// Round to reasonable precision
	recommendedCores = math.Round(recommendedCores*1000) / 1000

	// Allow larger changes for severely mis-provisioned workloads
	if currentLimit != nil {
		current := *currentLimit
		// If current is 0, always recommend (no limit set)
		if current > 0 {
			diff := math.Abs(recommendedCores - current)
			changePercent := diff / current

			// For severely mis-provisioned workloads, use a lower threshold
			threshold := 0.1
			if changePercent > 1.0 {
				// More than 100% difference - severely mis-provisioned
				threshold = 0.05
			}

			if changePercent < threshold {
				// Change is too small, consider it optimal
				return nil, nil
			}
		}
	}

	return &recommendedCores, nil
}

// Calculates memory request recommendation based on utilization analysis
func CalculateMemoryRequestRecommendation(
	currentRequest *float64,
	utilization MemoryUtilization,
	provisioning MemoryProvisioning,
	stability StabilityResult,
	config RecommendationConfig,
) (*float64, error) {
	// Adjust confidence threshold based on efficiency
	effectiveThreshold := config.MinConfidenceThreshold
	if provisioning.Efficiency < 0.3 {
		effectiveThreshold = effectiveThreshold * 0.8
	}

	// If we don't have enough confidence, don't recommend
	if provisioning.Confidence < effectiveThreshold {
		return nil, nil
	}

	// Start with P95 as baseline (steady-state usage)
	percentileP95 := utilization.Stats.Percentile.P95
	percentileP99 := utilization.Stats.Percentile.P99
	peakUsage := utilization.Stats.Max

	// Calculate coefficient of variation
	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	// Detect bursty workloads and adjust percentile
	adjustedPercentile, isBursty := detectAndAdjustBurstyWorkload(
		percentileP95, percentileP99, peakUsage, config.BurstThreshold,
	)

	// Apply memory pressure factor: if OOM detected, use peak instead of percentile
	if stability.MemoryOOM > 0 {
		adjustedPercentile = math.Max(adjustedPercentile, peakUsage)
	}

	// Calculate memory pressure factor
	pressureFactor := calculateMemoryPressureFactor(
		stability.MemoryOOM,
		stability.MemoryFailCnt,
		stability.MemoryPressure,
	)

	// Calculate recommended request based on mode
	var recommendedBytes float64
	switch config.Mode {
	case ModeCostOptimized:
		// Cost-optimized mode: use adjusted percentile
		baseMargin := config.CostOptimizedRequestMargin
		safetyMargin := calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			isBursty,
		)
		recommendedBytes = adjustedPercentile * safetyMargin * pressureFactor
	case ModeGuaranteed:
		// Guaranteed mode: use peak usage
		baseMargin := config.RequestSafetyMargin * config.GuaranteedMarginMultiplier
		safetyMargin := calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			isBursty,
		)
		recommendedBytes = peakUsage * safetyMargin * pressureFactor
	default:
		// Burstable mode: use adjusted percentile
		baseMargin := config.RequestSafetyMargin // Standard base
		safetyMargin := calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			isBursty,
		)
		recommendedBytes = adjustedPercentile * safetyMargin * pressureFactor
	}

	// Validate: ensure recommended accounts for actual usage patterns
	// Only use Mean if it's higher than the recommendation
	if utilization.Stats.Mean > 0 && recommendedBytes < utilization.Stats.Mean {
		meanBased := utilization.Stats.Mean * 1.15
		if meanBased > recommendedBytes {
			recommendedBytes = meanBased
		}
	}

	// Apply minimum
	if recommendedBytes < float64(config.MinMemoryRequest) {
		recommendedBytes = float64(config.MinMemoryRequest)
	}

	// Round to nearest byte
	recommendedBytes = math.Round(recommendedBytes)

	// Allow larger changes for severely mis-provisioned workloads
	if currentRequest != nil {
		current := *currentRequest
		// If current is 0, always recommend (no request set)
		if current > 0 {
			diff := math.Abs(recommendedBytes - current)
			changePercent := diff / current

			// For severely mis-provisioned workloads, use a lower threshold
			threshold := 0.1
			if changePercent > 1.0 {
				// More than 100% difference - severely mis-provisioned
				threshold = 0.05
			}

			if changePercent < threshold {
				// Change is too small, consider it optimal
				return nil, nil
			}
		}
	}

	return &recommendedBytes, nil
}

// Calculates memory limit recommendation based on utilization analysis
func CalculateMemoryLimitRecommendation(
	currentLimit *float64,
	utilization MemoryUtilization,
	provisioning MemoryProvisioning,
	stability StabilityResult,
	config RecommendationConfig,
	recommendedRequest *float64,
) (*float64, error) {
	// Adjust confidence threshold based on efficiency
	effectiveThreshold := config.MinConfidenceThreshold
	if provisioning.Efficiency < 0.3 {
		effectiveThreshold = effectiveThreshold * 0.8
	}

	// If we don't have enough confidence, don't recommend
	if provisioning.Confidence < effectiveThreshold {
		return nil, nil
	}

	// Use peak (max) for limit recommendation
	peakUsage := utilization.Stats.Max

	// Calculate coefficient of variation
	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	// Calculate memory pressure factor
	pressureFactor := calculateMemoryPressureFactor(
		stability.MemoryOOM,
		stability.MemoryFailCnt,
		stability.MemoryPressure,
	)

	// Calculate safety margin based on mode
	var safetyMargin float64
	switch config.Mode {
	case ModeCostOptimized:
		// Cost-optimized mode: minimal base margin
		baseMargin := config.CostOptimizedLimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false, // Limits don't need burst detection
		)
	case ModeGuaranteed:
		// Guaranteed mode: higher base margin
		baseMargin := config.LimitSafetyMargin * config.GuaranteedMarginMultiplier
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false, // Limits don't need burst detection
		)
	default:
		// Burstable mode: standard base margin
		baseMargin := config.LimitSafetyMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false, // Limits don't need burst detection
		)
	}

	// Calculate recommended limit
	recommendedBytes := peakUsage * safetyMargin * pressureFactor

	// Apply minimum
	if recommendedBytes < float64(config.MinMemoryRequest) {
		recommendedBytes = float64(config.MinMemoryRequest)
	}

	// Ensure limit is at least equal to recommended request
	if recommendedRequest != nil && recommendedBytes < *recommendedRequest {
		recommendedBytes = *recommendedRequest
	}

	// Round to nearest byte
	recommendedBytes = math.Round(recommendedBytes)

	// Allow larger changes for severely mis-provisioned workloads
	if currentLimit != nil {
		current := *currentLimit
		// If current is 0, always recommend (no limit set)
		if current > 0 {
			diff := math.Abs(recommendedBytes - current)
			changePercent := diff / current

			// For severely mis-provisioned workloads, use a lower threshold
			threshold := 0.1
			if changePercent > 1.0 {
				// More than 100% difference - severely mis-provisioned
				threshold = 0.05
			}

			if changePercent < threshold {
				// Change is too small, consider it optimal
				return nil, nil
			}
		}
	}

	return &recommendedBytes, nil
}

// Calculates CPU throttling pressure factor using gradual scaling
func calculateCPUThrottlingPressureFactor(throttling float64) float64 {
	if throttling <= 0 {
		return 1.0
	}
	if throttling < 0.01 {
		// Very small throttling (0-1%): very gentle scaling from 1.0 to 1.01
		return 1.0 + throttling*1.0
	}
	if throttling < 0.1 {
		// Minor throttling (1% to 10%): moderate scaling from 1.01 to 1.6
		return 1.01 + (throttling-0.01)/0.09*0.59
	}
	// Severe throttling (>= 10%): aggressive scaling from 1.6, scales linearly to 100% (6.1x at 100%)
	return 1.6 + (throttling-0.1)*5.0
}

// Calculates CPU pressure factor using gradual scaling
func calculateCPUPressureFactor(pressure float64) float64 {
	if pressure <= 0 {
		return 1.0
	}
	if pressure < 0.2 {
		// Low pressure (0-20%): gentle scaling from 1.0 to 1.05
		return 1.0 + pressure*0.25
	}
	// Higher pressure (>= 20%): moderate scaling from 1.05, scales linearly to 100% (3.05x at 100%)
	return 1.05 + (pressure-0.2)*2.5
}

// Calculates memory pressure factor using gradual scaling
func calculateMemoryPressureFactor(oom float64, failCnt float64, pressure float64) float64 {
	pressureFactor := 1.0

	// OOM events scale based on count
	if oom > 0 {
		// Standard: 1 OOM = 2.0x, 2 OOM = 2.5x, 3 OOM = 3.0x, 4+ OOM = 3.5x
		extraPerOOM := math.Max(0, (oom-1)*0.5)
		pressureFactor = 2.0 + math.Min(1.5, extraPerOOM)
	} else {
		// No OOM: consider allocation failures
		if failCnt > 0 {
			// Allocation failures: moderate boost from 1.0 to 1.25
			pressureFactor = 1.25
		}

		// Memory pressure
		if pressure > 0 {
			if pressure < 0.1 {
				// Low pressure (0-10%): gentle scaling from 1.0 to 1.05
				pressureBoost := pressure * 0.5
				pressureFactor = math.Max(pressureFactor, 1.0+pressureBoost)
			} else {
				// Higher pressure (>= 10%): moderate scaling from 1.05, scales linearly to 100% (3.075x at 100%)
				pressureBoost := 1.05 + (pressure-0.1)*2.25
				pressureFactor = math.Max(pressureFactor, pressureBoost)
			}
		}
	}

	return pressureFactor
}

// Detects bursty workload and adjusts percentile accordingly
func detectAndAdjustBurstyWorkload(
	percentileP95, percentileP99, peakUsage float64,
	burstThreshold float64,
) (adjustedPercentile float64, isBursty bool) {
	baselineP95 := percentileP95
	adjustedPercentile = percentileP95

	// Detect bursty workloads: if Max is significantly higher than percentile
	if percentileP95 > 0 && peakUsage > percentileP95*burstThreshold {
		isBursty = true
		// Use a higher percentile or weighted approach
		if percentileP99 > percentileP95*1.5 {
			adjustedPercentile = percentileP99
			if peakUsage > adjustedPercentile*burstThreshold {
				adjustedPercentile = calculateWeightedPercentile(adjustedPercentile, peakUsage, baselineP95)
			}
		} else {
			weightedPercentile := calculateWeightedPercentile(percentileP95, peakUsage, baselineP95)

			// Validate: if weighted result is still too extreme (> 5x P95), use P99 instead
			if weightedPercentile > percentileP95*5.0 {
				// Use P99 if available, otherwise use the weighted result
				if percentileP99 > 0 {
					adjustedPercentile = percentileP99
					if peakUsage > adjustedPercentile*burstThreshold {
						adjustedPercentile = calculateWeightedPercentile(adjustedPercentile, peakUsage, baselineP95)
					}
				} else {
					adjustedPercentile = weightedPercentile
				}
			} else {
				adjustedPercentile = weightedPercentile
			}
		}
	}

	return adjustedPercentile, isBursty
}

// Calculates dynamic safety margin adjustments based on utilization patterns
func calculateDynamicSafetyMargin(
	baseMargin float64,
	trend TrendResult,
	cv float64,
	anomalyCount int,
	isBursty bool,
) float64 {
	safetyMargin := baseMargin

	// Adjust safety margin based on trend
	if trend.Direction == DirectionIncreasing && trend.Strength > 0.5 {
		// Increasing trend with strong signal = add extra headroom
		safetyMargin *= 1.1
	} else if trend.Direction == DirectionDecreasing && trend.Strength > 0.5 {
		// Decreasing trend = can be slightly less conservative
		safetyMargin *= 0.95
	}

	// Adjust safety margin based on variance
	if cv > 0.5 {
		// High variance = more conservative
		safetyMargin *= 1.15
	} else if cv < 0.2 && cv > 0 {
		// Low variance = can be more precise
		safetyMargin *= 0.98
	}

	// Adjust safety margin based on anomalies
	if anomalyCount > 8 {
		safetyMargin *= 1.1
	}

	// Add safety margin for bursty workloads
	if isBursty {
		safetyMargin *= 1.15
	}

	return safetyMargin
}

// Calculates the maximum weight to use for weighted percentile calculation based on gap ratio
func calculateMaxWeightForGap(gapRatio float64) float64 {
	if gapRatio > 50.0 {
		// Very extreme gap (>50x): use 80% of Max
		return 0.8
	} else if gapRatio > 20.0 {
		// Extreme gap (20-50x): use 70% of Max
		return 0.7
	} else if gapRatio > 10.0 {
		// Very large gap (10-20x): use 60% of Max
		return 0.6
	} else if gapRatio > 5.0 {
		// Large gap (5-10x): use 50% of Max
		return 0.5
	} else {
		// Moderate gap (2-5x): use 45% of Max
		return 0.45
	}
}

// Calculates weighted percentile using dynamic weighting based on gap severity
func calculateWeightedPercentile(percentile, peakUsage, baselineP95 float64) float64 {
	gapRatio := peakUsage / baselineP95
	maxWeight := calculateMaxWeightForGap(gapRatio)
	percentileWeight := 1.0 - maxWeight
	return percentile*percentileWeight + (peakUsage-percentile)*maxWeight
}

// Calculates overall confidence score from CPU and memory provisioning
func calculateOverallConfidence(
	cpuProvisioning CPUProvisioning,
	memoryProvisioning MemoryProvisioning,
) float64 {
	// Weighted average of CPU and memory confidence
	// Equal weight
	cpuWeight := 0.5
	memoryWeight := 0.5

	overallConfidence := (cpuProvisioning.Confidence * cpuWeight) + (memoryProvisioning.Confidence * memoryWeight)

	// Ensure it's between 0 and 1
	overallConfidence = math.Max(0.0, math.Min(1.0, overallConfidence))

	return overallConfidence
}

// Finalizes request and limit recommendations based on mode and constraints.
func finalizeResourceRecommendations(
	recommendedRequest *float64,
	recommendedLimit *float64,
	currentRequest *float64,
	currentLimit *float64,
	mode RecommendationMode,
) (*float64, *float64) {
	// For Guaranteed mode, both request and limit MUST be equal
	if mode == ModeGuaranteed {
		if recommendedRequest == nil && recommendedLimit == nil {
			return nil, nil
		}

		var maxValue float64
		if recommendedRequest != nil && recommendedLimit != nil {
			maxValue = max(*recommendedRequest, *recommendedLimit)
		} else if recommendedRequest != nil {
			maxValue = *recommendedRequest
		} else {
			maxValue = *recommendedLimit
		}

		return &maxValue, &maxValue
	}

	// If no limit recommendation but we have a request
	if recommendedLimit == nil && recommendedRequest != nil {
		if currentLimit != nil && *currentLimit >= *recommendedRequest {
			return recommendedRequest, currentLimit
		}
		// Recommend a limit equal to request
		return recommendedRequest, recommendedRequest
	}

	// If no request but we have a limit
	if recommendedRequest == nil && recommendedLimit != nil {
		// If recommended limit is less than current request, adjust limit
		if currentRequest != nil && *recommendedLimit < *currentRequest {
			return currentRequest, currentRequest
		}
		if currentRequest != nil {
			return currentRequest, recommendedLimit
		}
		// No current request: use recommended limit for both
		return recommendedLimit, recommendedLimit
	}

	return recommendedRequest, recommendedLimit
}

// Formats CPU cores as Kubernetes resource quantity string
func formatCPUQuantity(cores float64) string {
	// Ensure non-negative
	cores = max(cores, 0)

	// For small values (< 1 core), use millicores (m)
	if cores < 1.0 {
		millicores := max(int64(cores*1000), 0)
		return fmt.Sprintf("%dm", millicores)
	}
	// For larger values, use cores with 3 decimal places
	return fmt.Sprintf("%.3f", cores)
}

// Formats memory bytes as Kubernetes resource quantity string
func formatMemoryQuantity(bytes int64) string {
	// Ensure non-negative
	bytes = max(bytes, 0)

	const (
		KiB = 1024
		MiB = 1024 * KiB
		GiB = 1024 * MiB
		TiB = 1024 * GiB
	)

	// Format to nearest unit
	switch {
	case bytes >= TiB:
		value := math.Round(float64(bytes) / TiB)
		return fmt.Sprintf("%dTi", int64(value))
	case bytes >= GiB:
		value := math.Round(float64(bytes) / GiB)
		return fmt.Sprintf("%dGi", int64(value))
	case bytes >= MiB:
		value := math.Round(float64(bytes) / MiB)
		return fmt.Sprintf("%dMi", int64(value))
	case bytes >= KiB:
		value := math.Round(float64(bytes) / KiB)
		return fmt.Sprintf("%dKi", int64(value))
	default:
		return fmt.Sprintf("%d", bytes)
	}
}
