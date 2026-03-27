package compute

import (
	"fmt"
	"math"
	"time"

	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/timeseries"
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
	// Safety margin multiplier for CPU requests (default: 1.25 = 25% headroom)
	CPURequestMargin float64
	// Safety margin multiplier for CPU limits (default: 1.35 = 35% headroom)
	CPULimitMargin float64
	// Safety margin multiplier for memory requests (default: 1.2 = 20% headroom)
	MemoryRequestMargin float64
	// Safety margin multiplier for memory limits (default: 1.3 = 30% headroom)
	MemoryLimitMargin float64
	// Safety margin for cost-optimized mode CPU requests (default: 1.15 = 15% headroom)
	CostOptimizedCPURequestMargin float64
	// Safety margin for cost-optimized mode CPU limits (default: 1.2 = 20% headroom)
	CostOptimizedCPULimitMargin float64
	// Safety margin for cost-optimized mode memory requests (default: 1.15 = 15% headroom)
	CostOptimizedMemoryRequestMargin float64
	// Safety margin for cost-optimized mode memory limits (default: 1.2 = 20% headroom)
	CostOptimizedMemoryLimitMargin float64
	// Minimum CPU request in cores (default: 0.01 = 10m)
	MinCPURequest float64
	// Minimum memory request in bytes (default: 64Mi)
	MinMemoryRequest int64
	// Minimum confidence threshold to generate recommendations (default: 0.8)
	MinConfidenceThreshold float64
	// Burst detection threshold: if Max > percentile * BurstThreshold, treat as burst workload
	BurstThreshold float64
	// Limit multiplier relative to request (default: 1.5)
	LimitMultiplier float64
	// Limit multiplier relative to request for cost-optimized mode (default: 1.2)
	CostOptimizedLimitMultiplier float64
	// Max reduction ratio per step: cost_optimized (default 0.5), burstable (0.4), guaranteed (0.3)
	CostOptimizedMaxReductionRatio float64
	BurstableMaxReductionRatio     float64
	GuaranteedMaxReductionRatio    float64
	// Max increase ratio per step (default 2.0)
	MaxIncreaseRatio float64
}

// Returns default recommendation configuration
func DefaultRecommendationConfig() RecommendationConfig {
	return RecommendationConfig{
		Mode:                             ModeBurstable,                                   // Default: optimize for efficiency
		CPURequestMargin:                 1.25,                                            // 25% headroom for CPU requests
		CPULimitMargin:                   1.35,                                            // 35% headroom for CPU limits
		MemoryRequestMargin:              1.2,                                             // 20% headroom for memory requests
		MemoryLimitMargin:                1.3,                                             // 30% headroom for memory limits
		CostOptimizedCPURequestMargin:    1.15,                                            // 15% headroom for cost-optimized CPU requests
		CostOptimizedCPULimitMargin:      1.2,                                             // 20% headroom for cost-optimized CPU limits
		CostOptimizedMemoryRequestMargin: 1.15,                                            // 15% headroom for cost-optimized memory requests
		CostOptimizedMemoryLimitMargin:   1.2,                                             // 20% headroom for cost-optimized memory limits
		MinCPURequest:                    0.01,                                            // 10m minimum
		MinMemoryRequest:                 64 * 1024 * 1024,                                // 64Mi minimum
		MinConfidenceThreshold:           config.AppConfig.Compute.MinConfidenceThreshold, // Minimum confidence (default: 0.8)
		BurstThreshold:                   1.6,                                             // Max > 1.6x percentile = burst workload
		LimitMultiplier:                  1.5,                                             // 1.5x limit
		CostOptimizedLimitMultiplier:     1.2,                                             // 1.2x limit for cost-optimized mode
		CostOptimizedMaxReductionRatio:   0.5,                                             // at most 50% reduction per step (floor 50% of current)
		BurstableMaxReductionRatio:       0.4,                                             // at most 40% reduction per step (floor 60% of current)
		GuaranteedMaxReductionRatio:      0.3,                                             // at most 30% reduction per step (floor 70% of current)
		MaxIncreaseRatio:                 2.0,                                             // at most 2x current per step
	}
}

// Validates recommendation configuration
func (config RecommendationConfig) Validate() error {
	if config.Mode != ModeCostOptimized && config.Mode != ModeBurstable && config.Mode != ModeGuaranteed {
		return fmt.Errorf("Mode must be one of %s, %s, or %s, got: %v", ModeCostOptimized, ModeBurstable, ModeGuaranteed, config.Mode)
	}
	if config.CPURequestMargin <= 0 {
		return fmt.Errorf("CPURequestMargin must be positive, got: %v", config.CPURequestMargin)
	}
	if config.CPULimitMargin <= 0 {
		return fmt.Errorf("CPULimitMargin must be positive, got: %v", config.CPULimitMargin)
	}
	if config.MemoryRequestMargin <= 0 {
		return fmt.Errorf("MemoryRequestMargin must be positive, got: %v", config.MemoryRequestMargin)
	}
	if config.MemoryLimitMargin <= 0 {
		return fmt.Errorf("MemoryLimitMargin must be positive, got: %v", config.MemoryLimitMargin)
	}
	if config.CostOptimizedCPURequestMargin <= 0 {
		return fmt.Errorf("CostOptimizedCPURequestMargin must be positive, got: %v", config.CostOptimizedCPURequestMargin)
	}
	if config.CostOptimizedCPULimitMargin <= 0 {
		return fmt.Errorf("CostOptimizedCPULimitMargin must be positive, got: %v", config.CostOptimizedCPULimitMargin)
	}
	if config.CostOptimizedMemoryRequestMargin <= 0 {
		return fmt.Errorf("CostOptimizedMemoryRequestMargin must be positive, got: %v", config.CostOptimizedMemoryRequestMargin)
	}
	if config.CostOptimizedMemoryLimitMargin <= 0 {
		return fmt.Errorf("CostOptimizedMemoryLimitMargin must be positive, got: %v", config.CostOptimizedMemoryLimitMargin)
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
	if config.LimitMultiplier <= 0 {
		return fmt.Errorf("LimitMultiplier must be positive, got: %v", config.LimitMultiplier)
	}
	if config.CostOptimizedLimitMultiplier <= 0 {
		return fmt.Errorf("CostOptimizedLimitMultiplier must be positive, got: %v", config.CostOptimizedLimitMultiplier)
	}
	if config.CostOptimizedMaxReductionRatio < 0 || config.CostOptimizedMaxReductionRatio > 1 {
		return fmt.Errorf("CostOptimizedMaxReductionRatio must be between 0 and 1, got: %v", config.CostOptimizedMaxReductionRatio)
	}
	if config.BurstableMaxReductionRatio < 0 || config.BurstableMaxReductionRatio > 1 {
		return fmt.Errorf("BurstableMaxReductionRatio must be between 0 and 1, got: %v", config.BurstableMaxReductionRatio)
	}
	if config.GuaranteedMaxReductionRatio < 0 || config.GuaranteedMaxReductionRatio > 1 {
		return fmt.Errorf("GuaranteedMaxReductionRatio must be between 0 and 1, got: %v", config.GuaranteedMaxReductionRatio)
	}
	if config.MaxIncreaseRatio < 1.0 || config.MaxIncreaseRatio > 5.0 {
		return fmt.Errorf("MaxIncreaseRatio must be between 1 and 5, got: %v", config.MaxIncreaseRatio)
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
	// If not first-time recommendation and we don't have enough confidence, don't recommend
	if !firstTime(currentRequest) && provisioning.Confidence < config.MinConfidenceThreshold {
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

	// For first-time recommendations, use max of adjusted percentile and peak usage
	if firstTime(currentRequest) {
		adjustedPercentile = max(adjustedPercentile, peakUsage)
	}

	// Calculate stress factor
	stressFactor := calculateCPUStressFactor(stability.CPUThrottling, stability.CPUPressure)

	// Calculate recommended request based on mode
	var recommendedCores float64
	switch config.Mode {
	case ModeCostOptimized:
		// Cost-optimized mode: use adjusted percentile
		baseMargin := config.CostOptimizedCPURequestMargin
		safetyMargin := calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			isBursty,
		)
		recommendedCores = adjustedPercentile * safetyMargin * stressFactor
	case ModeGuaranteed:
		// Guaranteed mode: use peak usage
		baseMargin := config.CPURequestMargin
		safetyMargin := calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			isBursty,
		)
		recommendedCores = peakUsage * safetyMargin * stressFactor
	default:
		// Burstable mode: use adjusted percentile
		baseMargin := config.CPURequestMargin
		safetyMargin := calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			isBursty,
		)
		recommendedCores = adjustedPercentile * safetyMargin * stressFactor
	}

	// Validation: ensure recommended accounts for actual usage patterns
	// Only use Mean if it's higher than the recommendation
	if utilization.Stats.Mean > 0 && recommendedCores < utilization.Stats.Mean {
		meanBased := utilization.Stats.Mean * 1.15
		recommendedCores = max(recommendedCores, meanBased)
	}

	// Apply minimum
	recommendedCores = max(recommendedCores, config.MinCPURequest)

	// If stress is detected and we have current requests
	// don't recommend less than current and account for cpu stress factor
	// Cost_optimized mode: don't apply stress floor when throttling and pressure are low
	if (stability.CPUThrottling > 0 || stability.CPUPressure > 0) && currentRequest != nil && *currentRequest > 0 {
		var minRequestFromStress float64
		if config.Mode == ModeCostOptimized &&
			stability.CPUThrottling <= 0.05 &&
			stability.CPUPressure <= 0.1 {
			// Don't reduce below current
			minRequestFromStress = *currentRequest
		} else {
			minRequestFromStress = *currentRequest * stressFactor
		}
		recommendedCores = max(recommendedCores, minRequestFromStress)
	}

	// Max reduction per step: don't recommend less than (1 - maxReductionRatio) of current request
	if currentRequest != nil && *currentRequest > 0 && recommendedCores < *currentRequest {
		floor := *currentRequest * (1 - maxReductionRatio(config))
		recommendedCores = max(recommendedCores, floor)
	}

	// Max increase per step: don't recommend more than current * MaxIncreaseRatio
	if currentRequest != nil && *currentRequest > 0 && recommendedCores > *currentRequest {
		ceiling := *currentRequest * config.MaxIncreaseRatio
		recommendedCores = min(recommendedCores, ceiling)
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
	currentRequest *float64,
) (*float64, error) {
	// If not first-time recommendation and we don't have enough confidence, don't recommend
	if !firstTime(currentLimit) && provisioning.Confidence < config.MinConfidenceThreshold {
		return nil, nil
	}

	// Use peak (max) for limit recommendation
	peakUsage := utilization.Stats.Max

	// Calculate coefficient of variation
	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	// Calculate stress factor
	stressFactor := calculateCPUStressFactor(stability.CPUThrottling, stability.CPUPressure)

	// Calculate safety margin based on mode
	var safetyMargin float64
	switch config.Mode {
	case ModeCostOptimized:
		// Cost-optimized mode: minimal base margin
		baseMargin := config.CostOptimizedCPULimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false, // Limits don't need burst detection
		)
	case ModeGuaranteed:
		// Guaranteed mode: standard base margin
		baseMargin := config.CPULimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false, // Limits don't need burst detection
		)
	default:
		// Burstable mode: standard base margin
		baseMargin := config.CPULimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false, // Limits don't need burst detection
		)
	}

	// Calculate limit from peak usage
	limitFromPeak := peakUsage * safetyMargin * stressFactor

	// Use recommended request when set, otherwise current request
	effectiveRequest := recommendedRequest
	if effectiveRequest == nil && currentRequest != nil && *currentRequest > 0 {
		effectiveRequest = currentRequest
	}

	// Calculate limit from request
	var limitFromRequest float64
	if effectiveRequest != nil && *effectiveRequest > 0 {
		var multiplier float64
		// When request is at minimum do not add multiplier to the limit
		if *effectiveRequest <= config.MinCPURequest {
			multiplier = 1.0
		} else {
			switch config.Mode {
			case ModeCostOptimized:
				multiplier = config.CostOptimizedLimitMultiplier
			case ModeGuaranteed:
				multiplier = config.LimitMultiplier
			default:
				// Burstable mode
				multiplier = config.LimitMultiplier
			}
		}
		limitFromRequest = *effectiveRequest * multiplier
	}

	// Maintain ratio and cover peak demand
	recommendedCores := max(limitFromPeak, limitFromRequest)

	// Apply minimum
	recommendedCores = max(recommendedCores, config.MinCPURequest)

	// Ensure limit is at least equal to request
	if effectiveRequest != nil && recommendedCores < *effectiveRequest {
		recommendedCores = *effectiveRequest
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
	analysisWindow time.Duration,
) (*float64, error) {
	// If not first-time recommendation and we don't have enough confidence, don't recommend
	if !firstTime(currentRequest) && provisioning.Confidence < config.MinConfidenceThreshold {
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

	// Apply memory stress factor: if OOM or failcnt detected, use max of adjusted percentile and peak usage
	if stability.MemoryOOM > 0 || stability.MemoryFailCnt > 0 {
		adjustedPercentile = max(adjustedPercentile, peakUsage)
	}

	// For first-time recommendations, use max of adjusted percentile and peak usage
	if firstTime(currentRequest) {
		adjustedPercentile = max(adjustedPercentile, peakUsage)
	}

	// Calculate memory stress factor
	stressFactor := calculateMemoryStressFactor(
		stability.MemoryOOM,
		stability.MemoryFailCnt,
		stability.MemoryPressure,
		analysisWindow,
	)

	// Calculate recommended request based on mode
	var recommendedBytes float64
	switch config.Mode {
	case ModeCostOptimized:
		// Cost-optimized mode: use adjusted percentile
		baseMargin := config.CostOptimizedMemoryRequestMargin
		safetyMargin := calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			isBursty,
		)
		recommendedBytes = adjustedPercentile * safetyMargin * stressFactor
	case ModeGuaranteed:
		// Guaranteed mode: use peak usage
		baseMargin := config.MemoryRequestMargin
		safetyMargin := calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			isBursty,
		)
		recommendedBytes = peakUsage * safetyMargin * stressFactor
	default:
		// Burstable mode: use adjusted percentile
		baseMargin := config.MemoryRequestMargin
		safetyMargin := calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			isBursty,
		)
		recommendedBytes = adjustedPercentile * safetyMargin * stressFactor
	}

	// Validate: ensure recommended accounts for actual usage patterns
	// Only use Mean if it's higher than the recommendation
	if utilization.Stats.Mean > 0 && recommendedBytes < utilization.Stats.Mean {
		meanBased := utilization.Stats.Mean * 1.15
		recommendedBytes = max(recommendedBytes, meanBased)
	}

	// Apply minimum
	recommendedBytes = max(recommendedBytes, float64(config.MinMemoryRequest))

	// If OOM or failcnt exists and we have current memory request
	// don't reduce below current request and account for pressure factor
	if (stability.MemoryOOM > 0 || stability.MemoryFailCnt > 0) && currentRequest != nil && *currentRequest > 0 {
		minRequestFromOOM := *currentRequest * stressFactor
		recommendedBytes = max(recommendedBytes, minRequestFromOOM)
	}

	// Max reduction per step: don't recommend less than (1 - maxReductionRatio) of current request
	if currentRequest != nil && *currentRequest > 0 && recommendedBytes < *currentRequest {
		floor := *currentRequest * (1 - maxReductionRatio(config))
		recommendedBytes = max(recommendedBytes, floor)
	}

	// Max increase per step: don't recommend more than current * MaxIncreaseRatio
	if currentRequest != nil && *currentRequest > 0 && recommendedBytes > *currentRequest {
		ceiling := float64(*currentRequest) * config.MaxIncreaseRatio
		recommendedBytes = min(recommendedBytes, ceiling)
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
	currentRequest *float64,
	analysisWindow time.Duration,
) (*float64, error) {
	// If not first-time recommendation and we don't have enough confidence, don't recommend
	if !firstTime(currentLimit) && provisioning.Confidence < config.MinConfidenceThreshold {
		return nil, nil
	}

	// Use peak (max) for limit recommendation
	peakUsage := utilization.Stats.Max

	// Calculate coefficient of variation
	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	// Calculate memory stress factor
	stressFactor := calculateMemoryStressFactor(
		stability.MemoryOOM,
		stability.MemoryFailCnt,
		stability.MemoryPressure,
		analysisWindow,
	)

	// Calculate safety margin based on mode
	var safetyMargin float64
	switch config.Mode {
	case ModeCostOptimized:
		// Cost-optimized mode: minimal base margin
		baseMargin := config.CostOptimizedMemoryLimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false, // Limits don't need burst detection
		)
	case ModeGuaranteed:
		// Guaranteed mode: standard base margin
		baseMargin := config.MemoryLimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false, // Limits don't need burst detection
		)
	default:
		// Burstable mode: standard base margin
		baseMargin := config.MemoryLimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false, // Limits don't need burst detection
		)
	}

	// Calculate limit from peak usage
	limitFromPeak := peakUsage * safetyMargin * stressFactor

	// Use recommended request when set, otherwise current request
	effectiveRequest := recommendedRequest
	if effectiveRequest == nil && currentRequest != nil && *currentRequest > 0 {
		effectiveRequest = currentRequest
	}

	// Calculate limit from request
	var limitFromRequest float64
	if effectiveRequest != nil && *effectiveRequest > 0 {
		var multiplier float64
		// When request is at minimum do not add multiplier to the limit
		if *effectiveRequest <= float64(config.MinMemoryRequest) {
			multiplier = 1.0
		} else {
			switch config.Mode {
			case ModeCostOptimized:
				multiplier = config.CostOptimizedLimitMultiplier
			case ModeGuaranteed:
				multiplier = config.LimitMultiplier
			default:
				// Burstable mode
				multiplier = config.LimitMultiplier
			}
		}
		limitFromRequest = *effectiveRequest * multiplier
	}

	// Maintain ratio and cover peak demand
	recommendedBytes := max(limitFromPeak, limitFromRequest)

	// Apply minimum
	recommendedBytes = max(recommendedBytes, float64(config.MinMemoryRequest))

	// Ensure limit is at least equal to request
	if effectiveRequest != nil && recommendedBytes < *effectiveRequest {
		recommendedBytes = *effectiveRequest
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

// Returns the max reduction ratio for the current mode.
func maxReductionRatio(config RecommendationConfig) float64 {
	switch config.Mode {
	case ModeCostOptimized:
		return config.CostOptimizedMaxReductionRatio
	case ModeGuaranteed:
		return config.GuaranteedMaxReductionRatio
	default:
		return config.BurstableMaxReductionRatio
	}
}

// Returns whether the current request/limits is not set (first-time recommendation).
func firstTime(current *float64) bool {
	return current == nil || *current == 0
}

// Calculates CPU stress factor
func calculateCPUStressFactor(throttling float64, pressure float64) float64 {
	throttlingFactor := calculateCPUThrottlingFactor(throttling)
	pressureFactor := calculatePSIPressureFactor(pressure)

	// Use the maximum of the two signals
	// This ensures we prioritize the most critical distress signal
	return max(throttlingFactor, pressureFactor)
}

// Calculates CPU throttling factor using gradual scaling
func calculateCPUThrottlingFactor(throttling float64) float64 {
	if throttling <= 0 {
		return 1.0
	}
	if throttling < 0.01 {
		// Very small throttling (0-1%): scale from 1.0 to 1.05
		return 1.0 + throttling*5.0
	}
	if throttling < 0.05 {
		// Minor throttling (1% to 5%): scale from 1.05 to 1.2
		return 1.05 + (throttling-0.01)*3.75
	}
	if throttling < 0.1 {
		// Moderate throttling (5% to 10%): scale from 1.2 to 1.25
		return 1.2 + (throttling-0.05)*1.0
	}
	if throttling < 0.2 {
		// Elevated throttling (10% to 20%): scale from 1.25 to 1.3
		return 1.25 + (throttling-0.1)*0.5
	}
	// High throttling (>= 20%): scale from 1.3
	// Capped at 1.8 (reached at ~60% throttling) to avoid extreme values
	val := 1.3 + (throttling-0.2)*1.25
	return min(val, 1.8)
}

// Calculates PSI pressure (cpu/memory) factor using gradual scaling
func calculatePSIPressureFactor(pressure float64) float64 {
	if pressure <= 0 {
		return 1.0
	}
	if pressure < 0.1 {
		// Low pressure (0-10%): gentle scaling from 1.0 to 1.05
		return 1.0 + pressure*0.5
	}
	// Higher pressure (>= 10%): moderate scaling from 1.05
	// Capped at 1.8 (reached at ~40% pressure) to avoid extreme values
	val := 1.05 + (pressure-0.1)*2.5
	return min(val, 1.8)
}

// Calculates memory stress factor
func calculateMemoryStressFactor(
	oom float64,
	failCnt float64,
	pressure float64,
	analysisWindow time.Duration,
) float64 {
	// Normalize event counters
	oomPerWeek := normalizeEventCounterToPerWeek(oom, analysisWindow)
	failCntPerWeek := normalizeEventCounterToPerWeek(failCnt, analysisWindow)

	// Calculate pressure factors
	oomFactor := calculateMemoryOOMRateFactor(oomPerWeek)
	failCntFactor := calculateMemoryFailRateFactor(failCntPerWeek)
	psiFactor := calculatePSIPressureFactor(pressure)

	// Use the maximum of the three signals
	// This ensures we prioritize the most critical distress signal
	stressFactor := max(failCntFactor, oomFactor)
	stressFactor = max(stressFactor, psiFactor)

	return stressFactor
}

// Calculates memory OOM factor from normalized OOM rate (events per week)
// Using gradual scaling
func calculateMemoryOOMRateFactor(oomPerWeek float64) float64 {
	if oomPerWeek <= 0 {
		return 1.0
	}
	factor := 1.0 + 0.8*(oomPerWeek/(oomPerWeek+1.5))
	return min(factor, 1.8)
}

// Calculates memory fail count factor from normalized fail rate (events per week)
// Using gradual scaling
func calculateMemoryFailRateFactor(failCntPerWeek float64) float64 {
	if failCntPerWeek <= 0 {
		return 1.0
	}
	factor := 1.0 + 0.4*(failCntPerWeek/(failCntPerWeek+3.0))
	return min(factor, 1.4)
}

// Normalizes event counter from a given analysis window into a per-week rate
func normalizeEventCounterToPerWeek(counter float64, analysisWindow time.Duration) float64 {
	if counter <= 0 || analysisWindow <= 0 {
		return 0
	}

	const secondsPerWeek = 7 * 24 * 60 * 60
	return (counter / analysisWindow.Seconds()) * secondsPerWeek
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
		if percentileP99 > percentileP95*1.4 {
			adjustedPercentile = percentileP99
			if peakUsage > adjustedPercentile*burstThreshold {
				adjustedPercentile = calculateWeightedPercentile(adjustedPercentile, peakUsage, baselineP95)
			}
		} else {
			weightedPercentile := calculateWeightedPercentile(percentileP95, peakUsage, baselineP95)

			// Validate: if weighted result is still too extreme (> 4.5x P95), use P99 instead
			if weightedPercentile > percentileP95*4.5 {
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
	trend timeseries.TrendResult,
	cv float64,
	anomalyCount int,
	isBursty bool,
) float64 {
	safetyMargin := baseMargin

	// Adjust safety margin based on trend
	if trend.Direction == timeseries.DirectionIncreasing && trend.Strength > 0.5 {
		// Increasing trend with strong signal = add extra headroom
		safetyMargin *= 1.1
	} else if trend.Direction == timeseries.DirectionDecreasing && trend.Strength > 0.5 {
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
		// Moderate gap (2-5x): use 40% of Max
		return 0.4
	}
}

// Calculates weighted percentile using dynamic weighting based on gap severity
func calculateWeightedPercentile(percentile, peakUsage, baselineP95 float64) float64 {
	gapRatio := peakUsage / baselineP95
	maxWeight := calculateMaxWeightForGap(gapRatio)
	percentileWeight := 1.0 - maxWeight
	return percentile*percentileWeight + (peakUsage-percentile)*maxWeight
}
