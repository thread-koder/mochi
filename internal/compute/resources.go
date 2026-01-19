package compute

import (
	"fmt"
	"math"

	"k8s.io/apimachinery/pkg/api/resource"
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
		Mode:                   ModeBurstable,    // Default: optimize for efficiency
		RequestSafetyMargin:    1.2,              // 20% headroom
		LimitSafetyMargin:      1.3,              // 30% headroom
		MinCPURequest:          0.01,             // 10m minimum
		MinMemoryRequest:       64 * 1024 * 1024, // 64Mi minimum
		MinConfidenceThreshold: 0.5,              // 50% minimum confidence
		BurstThreshold:         2.0,              // Max > 2x percentile = burst workload
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
	percentile := utilization.Stats.Percentile.P95
	baselineP95 := percentile
	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	// Detect bursty workloads: if Max is significantly higher than percentile, it indicates occasional spikes
	peakUsage := utilization.Stats.Max
	isBurstWorkload := false
	if percentile > 0 && peakUsage > percentile*config.BurstThreshold {
		isBurstWorkload = true
		// Use a higher percentile or weighted approach
		// Check if P99 is much higher than P95
		if utilization.Stats.Percentile.P99 > percentile*1.5 {
			percentile = utilization.Stats.Percentile.P99
			// Check if Max is still significantly higher than P99
			if peakUsage > percentile*config.BurstThreshold {
				percentile = calculateWeightedPercentile(percentile, peakUsage, baselineP95)
			}
		} else {
			weightedPercentile := calculateWeightedPercentile(percentile, peakUsage, baselineP95)

			// Validate: if weighted result is still too extreme (> 5x P95), use P99 instead
			if weightedPercentile > percentile*5.0 {
				// Use P99 if available, otherwise use the weighted result
				if utilization.Stats.Percentile.P99 > 0 {
					percentile = utilization.Stats.Percentile.P99
					// Check if Max is still higher than P99
					if peakUsage > percentile*config.BurstThreshold {
						percentile = calculateWeightedPercentile(percentile, peakUsage, baselineP95)
					}
				} else {
					percentile = weightedPercentile
				}
			} else {
				percentile = weightedPercentile
			}
		}
	}

	// Calculate recommended request based on mode
	var recommendedCores float64
	switch config.Mode {
	case ModeGuaranteed:
		// Guaranteed mode: use base safety margin only
		recommendedCores = peakUsage * config.RequestSafetyMargin
	case ModeCostOptimized:
		// Cost-optimized mode: use percentile with minimal safety margin (1.1x)
		recommendedCores = percentile * 1.1
	default:
		// Burstable mode: calculate safety margin with all dynamic adjustments
		safetyMargin := config.RequestSafetyMargin

		// Adjust safety margin based on trend
		if utilization.Trend.Direction == DirectionIncreasing && utilization.Trend.Strength > 0.5 {
			// Increasing trend with strong signal = add extra headroom
			safetyMargin *= 1.1
		} else if utilization.Trend.Direction == DirectionDecreasing && utilization.Trend.Strength > 0.5 {
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
		if utilization.Anomalies.AnomalyCount > 10 {
			safetyMargin *= 1.1
		}

		// Add safety margin for bursty workloads
		if isBurstWorkload {
			safetyMargin *= 1.15
		}

		// Burstable mode: use percentile directly with safety margin
		recommendedCores = percentile * safetyMargin
	}

	// Validation: ensure recommended accounts for actual usage patterns
	// Only use Mean if it's higher than the recommendation
	if utilization.Stats.Mean > 0 && recommendedCores < utilization.Stats.Mean {
		meanBased := utilization.Stats.Mean * 1.1
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

	// Calculate safety margin based on mode
	var safetyMargin float64
	if config.Mode == ModeCostOptimized {
		// Cost-optimized mode: minimal safety margin (1.1x)
		safetyMargin = 1.1
	} else {
		// Burstable/Guaranteed mode: base + trend only
		safetyMargin = config.LimitSafetyMargin
		if utilization.Trend.Direction == DirectionIncreasing && utilization.Trend.Strength > 0.5 {
			// Increasing trend = add extra headroom for future growth
			safetyMargin *= 1.1
		}
	}

	// Calculate recommended limit
	recommendedCores := peakUsage * safetyMargin

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
	percentile := utilization.Stats.Percentile.P95
	baselineP95 := percentile
	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	// Detect bursty workloads: if Max is significantly higher than percentile, it indicates occasional spikes
	peakUsage := utilization.Stats.Max
	isBurstWorkload := false
	if percentile > 0 && peakUsage > percentile*config.BurstThreshold {
		isBurstWorkload = true
		// Use a higher percentile or weighted approach
		// Check if P99 is much higher than P95
		if utilization.Stats.Percentile.P99 > percentile*1.5 {
			percentile = utilization.Stats.Percentile.P99
			// Check if Max is still significantly higher than P99
			if peakUsage > percentile*config.BurstThreshold {
				percentile = calculateWeightedPercentile(percentile, peakUsage, baselineP95)
			}
		} else {
			weightedPercentile := calculateWeightedPercentile(percentile, peakUsage, baselineP95)

			// Validate: if weighted result is still too extreme (> 5x P95), use P99 instead
			if weightedPercentile > percentile*5.0 {
				// Use P99 if available, otherwise use the weighted result
				if utilization.Stats.Percentile.P99 > 0 {
					percentile = utilization.Stats.Percentile.P99
					// Check if Max is still higher than P99
					if peakUsage > percentile*config.BurstThreshold {
						percentile = calculateWeightedPercentile(percentile, peakUsage, baselineP95)
					}
				} else {
					percentile = weightedPercentile
				}
			} else {
				percentile = weightedPercentile
			}
		}
	}

	// Calculate recommended request based on mode
	var recommendedBytes float64
	switch config.Mode {
	case ModeGuaranteed:
		// Guaranteed mode: use base safety margin only
		recommendedBytes = peakUsage * config.RequestSafetyMargin * 1.1
	case ModeCostOptimized:
		// Cost-optimized mode: use percentile with minimal safety margin (1.1x)
		recommendedBytes = percentile * 1.1
	default:
		// Burstable mode: calculate safety margin with all dynamic adjustments
		safetyMargin := config.RequestSafetyMargin * 1.1

		// Adjust safety margin based on trend
		if utilization.Trend.Direction == DirectionIncreasing && utilization.Trend.Strength > 0.5 {
			// Increasing trend with strong signal = add extra headroom
			safetyMargin *= 1.1
		} else if utilization.Trend.Direction == DirectionDecreasing && utilization.Trend.Strength > 0.5 {
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
		if utilization.Anomalies.AnomalyCount > 10 {
			safetyMargin *= 1.1
		}

		// Add safety margin for bursty workloads
		if isBurstWorkload {
			safetyMargin *= 1.15
		}

		// Burstable mode: use percentile directly with safety margin
		recommendedBytes = percentile * safetyMargin
	}

	// Validate: ensure recommended accounts for actual usage patterns
	// Only use Mean if it's higher than the recommendation
	if utilization.Stats.Mean > 0 && recommendedBytes < utilization.Stats.Mean {
		meanBased := utilization.Stats.Mean * 1.1
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

	// Calculate safety margin based on mode
	var safetyMargin float64
	if config.Mode == ModeCostOptimized {
		// Cost-optimized mode: minimal safety margin (1.1x)
		safetyMargin = 1.1
	} else {
		// Burstable/Guaranteed mode: base + trend only
		safetyMargin = config.LimitSafetyMargin * 1.1
		if utilization.Trend.Direction == DirectionIncreasing && utilization.Trend.Strength > 0.5 {
			// Increasing trend = add extra headroom for future growth
			safetyMargin *= 1.1
		}
	}

	// Calculate recommended limit
	recommendedBytes := peakUsage * safetyMargin

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

// Calculates the maximum weight to use for weighted percentile calculation based on gap ratio
func calculateMaxWeightForGap(gapRatio float64) float64 {
	if gapRatio > 50.0 {
		// Very extreme gap (>50x): use 70% of Max to prevent throttling
		return 0.7
	} else if gapRatio > 20.0 {
		// Extreme gap (20-50x): use 60% of Max to prevent throttling
		return 0.6
	} else if gapRatio > 10.0 {
		// Very large gap (10-20x): use 50% of Max
		return 0.5
	} else if gapRatio > 5.0 {
		// Large gap (5-10x): use 40% of Max
		return 0.4
	} else {
		// Moderate gap (2-5x): use 30% of Max
		return 0.3
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
func CalculateOverallConfidence(
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

// Ensures limit is greater than or equal to request
func ensureLimitGreaterThanRequestValue(
	recommendedLimit *float64,
	recommendedRequest *float64,
	currentLimit *float64,
	currentRequest *float64,
) *float64 {
	// Determine the effective request value (prefer recommended, fall back to current)
	var effectiveRequest *float64
	if recommendedRequest != nil {
		effectiveRequest = recommendedRequest
	} else if currentRequest != nil {
		effectiveRequest = currentRequest
	}

	// If no request value exists, return limit as-is
	if effectiveRequest == nil {
		return recommendedLimit
	}

	// If no limit recommendation, but we have a request, we need to recommend a limit
	if recommendedLimit == nil {
		// If current limit exists and is >= request, no need to recommend
		if currentLimit != nil && *currentLimit >= *effectiveRequest {
			return nil
		}
		// Otherwise, recommend a limit equal to request (minimum)
		limitValue := *effectiveRequest
		return &limitValue
	}

	// Ensure recommended limit is at least equal to effective request
	if *recommendedLimit < *effectiveRequest {
		limitValue := *effectiveRequest
		return &limitValue
	}

	return recommendedLimit
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
	// Use Kubernetes resource.Quantity to format properly
	qty := resource.NewQuantity(bytes, resource.BinarySI)
	return qty.String()
}
