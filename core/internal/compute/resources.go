package compute

import (
	"fmt"
	"math"
	"time"

	"github.com/thread_koder/mochi/core/internal/timeseries"
)

// RecommendationMode selects sizing behavior: cost_optimized, burstable, or guaranteed.
type RecommendationMode string

const (
	ModeCostOptimized RecommendationMode = "cost_optimized"
	ModeBurstable     RecommendationMode = "burstable"
	ModeGuaranteed    RecommendationMode = "guaranteed"
)

// RecommendationConfig holds margins, burst and stress tuning, confidence gates, and per-step change limits.
type RecommendationConfig struct {
	Mode RecommendationMode

	CPURequestMargin    float64
	CPULimitMargin      float64
	MemoryRequestMargin float64
	MemoryLimitMargin   float64

	CostOptimizedCPURequestMargin    float64
	CostOptimizedCPULimitMargin      float64
	CostOptimizedMemoryRequestMargin float64
	CostOptimizedMemoryLimitMargin   float64

	MinConfidenceThreshold float64
	BurstThreshold         float64

	LimitMultiplier              float64
	CostOptimizedLimitMultiplier float64

	CostOptimizedMaxReductionRatio float64
	BurstableMaxReductionRatio     float64
	GuaranteedMaxReductionRatio    float64
	MaxIncreaseRatio               float64
}

func DefaultRecommendationConfig() RecommendationConfig {
	return RecommendationConfig{
		Mode:                             ModeBurstable,
		CPURequestMargin:                 1.25,
		CPULimitMargin:                   1.35,
		MemoryRequestMargin:              1.2,
		MemoryLimitMargin:                1.3,
		CostOptimizedCPURequestMargin:    1.15,
		CostOptimizedCPULimitMargin:      1.2,
		CostOptimizedMemoryRequestMargin: 1.15,
		CostOptimizedMemoryLimitMargin:   1.2,
		MinConfidenceThreshold:           0.8,
		BurstThreshold:                   1.6,
		LimitMultiplier:                  1.5,
		CostOptimizedLimitMultiplier:     1.2,
		CostOptimizedMaxReductionRatio:   0.5,
		BurstableMaxReductionRatio:       0.4,
		GuaranteedMaxReductionRatio:      0.3,
		MaxIncreaseRatio:                 2.0,
	}
}

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

func CalculateCPURequestRecommendation(
	currentRequest *float64,
	utilization CPUUtilization,
	provisioning CPUProvisioning,
	stability StabilityResult,
	config RecommendationConfig,
) *float64 {
	if provisioning.Confidence < config.MinConfidenceThreshold {
		return nil
	}

	percentileP95 := utilization.Stats.Percentile.P95
	percentileP99 := utilization.Stats.Percentile.P99
	peakUsage := utilization.Stats.Max

	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	adjustedPercentile, isBursty := detectAndAdjustBurstyWorkload(
		percentileP95, percentileP99, peakUsage, config.BurstThreshold,
	)

	if firstTime(currentRequest) {
		adjustedPercentile = max(adjustedPercentile, peakUsage)
	}

	stressFactor := calculateCPUStressFactor(stability.CPUThrottling, stability.CPUPressure)

	var recommendedCores float64
	switch config.Mode {
	case ModeCostOptimized:
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

	if utilization.Stats.Mean > 0 && recommendedCores < utilization.Stats.Mean {
		meanBased := utilization.Stats.Mean * 1.15
		recommendedCores = max(recommendedCores, meanBased)
	}

	recommendedCores = max(recommendedCores, MinCPURequestCores)

	if (stability.CPUThrottling > 0 || stability.CPUPressure > 0) && currentRequest != nil && *currentRequest > 0 {
		var minRequestFromStress float64
		if config.Mode == ModeCostOptimized &&
			stability.CPUThrottling <= 0.05 &&
			stability.CPUPressure <= 0.1 {
			minRequestFromStress = *currentRequest
		} else {
			minRequestFromStress = *currentRequest * stressFactor
		}
		recommendedCores = max(recommendedCores, minRequestFromStress)
	}

	if currentRequest != nil && *currentRequest > 0 && recommendedCores < *currentRequest {
		floor := *currentRequest * (1 - maxReductionRatio(config))
		recommendedCores = max(recommendedCores, floor)
	}

	if currentRequest != nil && *currentRequest > 0 && recommendedCores > *currentRequest {
		ceiling := *currentRequest * config.MaxIncreaseRatio
		recommendedCores = min(recommendedCores, ceiling)
	}

	recommendedCores = math.Round(recommendedCores*1000) / 1000

	if currentRequest != nil {
		current := *currentRequest
		if current > 0 {
			diff := math.Abs(recommendedCores - current)
			changePercent := diff / current

			if changePercent < 0.1 {
				return nil
			}
		}
	}

	return &recommendedCores
}

func CalculateCPULimitRecommendation(
	currentLimit *float64,
	utilization CPUUtilization,
	provisioning CPUProvisioning,
	stability StabilityResult,
	config RecommendationConfig,
	recommendedRequest *float64,
	currentRequest *float64,
) *float64 {
	if provisioning.Confidence < config.MinConfidenceThreshold {
		return nil
	}

	peakUsage := utilization.Stats.Max

	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	stressFactor := calculateCPUStressFactor(stability.CPUThrottling, stability.CPUPressure)

	var safetyMargin float64
	switch config.Mode {
	case ModeCostOptimized:
		baseMargin := config.CostOptimizedCPULimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false,
		)
	case ModeGuaranteed:
		baseMargin := config.CPULimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false,
		)
	default:
		baseMargin := config.CPULimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false,
		)
	}

	limitFromPeak := peakUsage * safetyMargin * stressFactor

	effectiveRequest := recommendedRequest
	if effectiveRequest == nil && currentRequest != nil && *currentRequest > 0 {
		effectiveRequest = currentRequest
	}

	var limitFromRequest float64
	if effectiveRequest != nil && *effectiveRequest > 0 {
		var multiplier float64
		if *effectiveRequest <= MinCPURequestCores {
			multiplier = 1.0
		} else {
			switch config.Mode {
			case ModeCostOptimized:
				multiplier = config.CostOptimizedLimitMultiplier
			case ModeGuaranteed:
				multiplier = config.LimitMultiplier
			default:
				multiplier = config.LimitMultiplier
			}
		}
		limitFromRequest = *effectiveRequest * multiplier
	}

	recommendedCores := max(limitFromPeak, limitFromRequest)

	recommendedCores = max(recommendedCores, MinCPURequestCores)

	if effectiveRequest != nil && recommendedCores < *effectiveRequest {
		recommendedCores = *effectiveRequest
	}

	recommendedCores = math.Round(recommendedCores*1000) / 1000

	if currentLimit != nil {
		current := *currentLimit
		if current > 0 {
			diff := math.Abs(recommendedCores - current)
			changePercent := diff / current

			if changePercent < 0.1 {
				return nil
			}
		}
	}

	return &recommendedCores
}

func CalculateMemoryRequestRecommendation(
	currentRequest *float64,
	utilization MemoryUtilization,
	provisioning MemoryProvisioning,
	stability StabilityResult,
	config RecommendationConfig,
	analysisWindow time.Duration,
) *float64 {
	if provisioning.Confidence < config.MinConfidenceThreshold {
		return nil
	}

	percentileP95 := utilization.Stats.Percentile.P95
	percentileP99 := utilization.Stats.Percentile.P99
	peakUsage := utilization.Stats.Max

	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	adjustedPercentile, isBursty := detectAndAdjustBurstyWorkload(
		percentileP95, percentileP99, peakUsage, config.BurstThreshold,
	)

	if stability.MemoryOOM > 0 || stability.MemoryFailCnt > 0 {
		adjustedPercentile = max(adjustedPercentile, peakUsage)
	}

	if firstTime(currentRequest) {
		adjustedPercentile = max(adjustedPercentile, peakUsage)
	}

	stressFactor := calculateMemoryStressFactor(
		stability.MemoryOOM,
		stability.MemoryFailCnt,
		stability.MemoryPressure,
		analysisWindow,
	)

	var recommendedBytes float64
	switch config.Mode {
	case ModeCostOptimized:
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

	if utilization.Stats.Mean > 0 && recommendedBytes < utilization.Stats.Mean {
		meanBased := utilization.Stats.Mean * 1.15
		recommendedBytes = max(recommendedBytes, meanBased)
	}

	recommendedBytes = max(recommendedBytes, float64(MinMemoryRequestBytes))

	if (stability.MemoryOOM > 0 || stability.MemoryFailCnt > 0) && currentRequest != nil && *currentRequest > 0 {
		minRequestFromOOM := *currentRequest * stressFactor
		recommendedBytes = max(recommendedBytes, minRequestFromOOM)
	}

	if currentRequest != nil && *currentRequest > 0 && recommendedBytes < *currentRequest {
		floor := *currentRequest * (1 - maxReductionRatio(config))
		recommendedBytes = max(recommendedBytes, floor)
	}

	if currentRequest != nil && *currentRequest > 0 && recommendedBytes > *currentRequest {
		ceiling := float64(*currentRequest) * config.MaxIncreaseRatio
		recommendedBytes = min(recommendedBytes, ceiling)
	}

	recommendedBytes = math.Round(recommendedBytes)

	if currentRequest != nil {
		current := *currentRequest
		if current > 0 {
			diff := math.Abs(recommendedBytes - current)
			changePercent := diff / current

			if changePercent < 0.1 {
				return nil
			}
		}
	}

	return &recommendedBytes
}

func CalculateMemoryLimitRecommendation(
	currentLimit *float64,
	utilization MemoryUtilization,
	provisioning MemoryProvisioning,
	stability StabilityResult,
	config RecommendationConfig,
	recommendedRequest *float64,
	currentRequest *float64,
	analysisWindow time.Duration,
) *float64 {
	if provisioning.Confidence < config.MinConfidenceThreshold {
		return nil
	}

	peakUsage := utilization.Stats.Max

	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	stressFactor := calculateMemoryStressFactor(
		stability.MemoryOOM,
		stability.MemoryFailCnt,
		stability.MemoryPressure,
		analysisWindow,
	)

	var safetyMargin float64
	switch config.Mode {
	case ModeCostOptimized:
		baseMargin := config.CostOptimizedMemoryLimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false,
		)
	case ModeGuaranteed:
		baseMargin := config.MemoryLimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false,
		)
	default:
		baseMargin := config.MemoryLimitMargin
		safetyMargin = calculateDynamicSafetyMargin(
			baseMargin,
			utilization.Trend,
			cv,
			utilization.Anomalies.AnomalyCount,
			false,
		)
	}

	limitFromPeak := peakUsage * safetyMargin * stressFactor

	effectiveRequest := recommendedRequest
	if effectiveRequest == nil && currentRequest != nil && *currentRequest > 0 {
		effectiveRequest = currentRequest
	}

	var limitFromRequest float64
	if effectiveRequest != nil && *effectiveRequest > 0 {
		var multiplier float64
		if *effectiveRequest <= float64(MinMemoryRequestBytes) {
			multiplier = 1.0
		} else {
			switch config.Mode {
			case ModeCostOptimized:
				multiplier = config.CostOptimizedLimitMultiplier
			case ModeGuaranteed:
				multiplier = config.LimitMultiplier
			default:
				multiplier = config.LimitMultiplier
			}
		}
		limitFromRequest = *effectiveRequest * multiplier
	}

	recommendedBytes := max(limitFromPeak, limitFromRequest)

	recommendedBytes = max(recommendedBytes, float64(MinMemoryRequestBytes))

	if effectiveRequest != nil && recommendedBytes < *effectiveRequest {
		recommendedBytes = *effectiveRequest
	}

	recommendedBytes = math.Round(recommendedBytes)

	if currentLimit != nil {
		current := *currentLimit
		if current > 0 {
			diff := math.Abs(recommendedBytes - current)
			changePercent := diff / current

			if changePercent < 0.1 {
				return nil
			}
		}
	}

	return &recommendedBytes
}

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

func firstTime(current *float64) bool {
	return current == nil || *current == 0
}

// calculateCPUStressFactor returns the larger of the throttling
// and PSI multipliers so either signal can raise headroom.
func calculateCPUStressFactor(throttling float64, pressure float64) float64 {
	throttlingFactor := calculateCPUThrottlingFactor(throttling)
	pressureFactor := calculatePSIPressureFactor(pressure)
	return max(throttlingFactor, pressureFactor)
}

func calculateCPUThrottlingFactor(throttling float64) float64 {
	if throttling <= 0 {
		return 1.0
	}
	if throttling < 0.01 {
		return 1.0 + throttling*5.0
	}
	if throttling < 0.05 {
		return 1.05 + (throttling-0.01)*3.75
	}
	if throttling < 0.1 {
		return 1.2 + (throttling-0.05)*1.0
	}
	if throttling < 0.2 {
		return 1.25 + (throttling-0.1)*0.5
	}
	val := 1.3 + (throttling-0.2)*1.25
	return min(val, 1.8)
}

func calculatePSIPressureFactor(pressure float64) float64 {
	if pressure <= 0 {
		return 1.0
	}
	if pressure < 0.1 {
		return 1.0 + pressure*0.5
	}
	val := 1.05 + (pressure-0.1)*2.5
	return min(val, 1.8)
}

// calculateMemoryStressFactor combines OOM rate, fail rate (both normalized per week),
// and PSI. The maximum signal drives headroom.
func calculateMemoryStressFactor(
	oom float64,
	failCnt float64,
	pressure float64,
	analysisWindow time.Duration,
) float64 {
	oomPerWeek := normalizeEventCounterToPerWeek(oom, analysisWindow)
	failCntPerWeek := normalizeEventCounterToPerWeek(failCnt, analysisWindow)

	oomFactor := calculateMemoryOOMRateFactor(oomPerWeek)
	failCntFactor := calculateMemoryFailRateFactor(failCntPerWeek)
	psiFactor := calculatePSIPressureFactor(pressure)

	stressFactor := max(failCntFactor, oomFactor)
	stressFactor = max(stressFactor, psiFactor)

	return stressFactor
}

func calculateMemoryOOMRateFactor(oomPerWeek float64) float64 {
	if oomPerWeek <= 0 {
		return 1.0
	}
	factor := 1.0 + 0.8*(oomPerWeek/(oomPerWeek+1.5))
	return min(factor, 1.8)
}

func calculateMemoryFailRateFactor(failCntPerWeek float64) float64 {
	if failCntPerWeek <= 0 {
		return 1.0
	}
	factor := 1.0 + 0.4*(failCntPerWeek/(failCntPerWeek+3.0))
	return min(factor, 1.4)
}

func normalizeEventCounterToPerWeek(counter float64, analysisWindow time.Duration) float64 {
	if counter <= 0 || analysisWindow <= 0 {
		return 0
	}

	const secondsPerWeek = 7 * 24 * 60 * 60
	return (counter / analysisWindow.Seconds()) * secondsPerWeek
}

// detectAndAdjustBurstyWorkload raises the baseline toward P99 or a weighted peak blend when max usage
// exceeds burstThreshold times P95.
// isBursty is used as an indicator that the percentile has been adjusted.
func detectAndAdjustBurstyWorkload(
	percentileP95, percentileP99, peakUsage float64,
	burstThreshold float64,
) (adjustedPercentile float64, isBursty bool) {
	baselineP95 := percentileP95
	adjustedPercentile = percentileP95

	if percentileP95 > 0 && peakUsage > percentileP95*burstThreshold {
		isBursty = true
		if percentileP99 > percentileP95*1.4 {
			adjustedPercentile = percentileP99
			if peakUsage > adjustedPercentile*burstThreshold {
				adjustedPercentile = calculateWeightedPercentile(adjustedPercentile, peakUsage, baselineP95)
			}
		} else {
			weightedPercentile := calculateWeightedPercentile(percentileP95, peakUsage, baselineP95)

			if weightedPercentile > percentileP95*4.5 {
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

func calculateDynamicSafetyMargin(
	baseMargin float64,
	trend timeseries.TrendResult,
	cv float64,
	anomalyCount int,
	isBursty bool,
) float64 {
	safetyMargin := baseMargin

	if trend.Direction == timeseries.DirectionIncreasing && trend.Strength > 0.5 {
		safetyMargin *= 1.1
	} else if trend.Direction == timeseries.DirectionDecreasing && trend.Strength > 0.5 {
		safetyMargin *= 0.95
	}

	if cv > 0.5 {
		safetyMargin *= 1.15
	} else if cv < 0.2 && cv > 0 {
		safetyMargin *= 0.98
	}

	if anomalyCount > 8 {
		safetyMargin *= 1.1
	}

	if isBursty {
		safetyMargin *= 1.15
	}

	return safetyMargin
}

func calculateMaxWeightForGap(gapRatio float64) float64 {
	if gapRatio > 50.0 {
		return 0.8
	} else if gapRatio > 20.0 {
		return 0.7
	} else if gapRatio > 10.0 {
		return 0.6
	} else if gapRatio > 5.0 {
		return 0.5
	}
	return 0.4
}

func calculateWeightedPercentile(percentile, peakUsage, baselineP95 float64) float64 {
	gapRatio := peakUsage / baselineP95
	maxWeight := calculateMaxWeightForGap(gapRatio)
	percentileWeight := 1.0 - maxWeight
	return percentile*percentileWeight + (peakUsage-percentile)*maxWeight
}
