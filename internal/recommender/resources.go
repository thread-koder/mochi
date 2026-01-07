package recommender

import (
	"fmt"
	"math"

	"github.com/thread_koder/mochi/internal/analyzer"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Represents recommended resource values
type ResourceRecommendation struct {
	CPURequest    *string // Recommended CPU request (e.g., "100m", "0.5")
	CPULimit      *string // Recommended CPU limit
	MemoryRequest *string // Recommended memory request (e.g., "128Mi", "1Gi")
	MemoryLimit   *string // Recommended memory limit
	Confidence    float64 // Overall confidence score (0-1)
}

// Represents the reason for a recommendation
type RecommendationReason string

const (
	ReasonOverProvisioned  RecommendationReason = "over_provisioned"
	ReasonUnderProvisioned RecommendationReason = "under_provisioned"
	ReasonOptimal          RecommendationReason = "optimal"
	ReasonNoData           RecommendationReason = "no_data"
)

// Represents the recommendation mode
type RecommendationMode string

const (
	ModeBurstable  RecommendationMode = "burstable"  // Default: optimize for efficiency, allow some throttling risk
	ModeGuaranteed RecommendationMode = "guaranteed" // Request = peak * safetyMargin (no throttling risk)
)

// Configuration for recommendation calculations
type RecommendationConfig struct {
	// Recommendation mode: burstable (optimize for efficiency) or guaranteed (no throttling risk)
	Mode RecommendationMode
	// Target utilization range for requests (default: 0.50-0.70 = 50-70%)
	// Only used in burstable mode
	TargetRequestUtilizationMin float64
	TargetRequestUtilizationMax float64
	// Safety margin multiplier for requests (default: 1.2 = 20% headroom)
	RequestSafetyMargin float64
	// Safety margin multiplier for limits based on peak (default: 1.3 = 30% headroom)
	LimitSafetyMargin float64
	// Minimum CPU request in cores (default: 0.01 = 10m)
	MinCPURequest float64
	// Minimum memory request in bytes (default: 64Mi)
	MinMemoryRequest int64
	// Minimum confidence threshold to generate recommendations (default: 0.3)
	MinConfidenceThreshold float64
	// Maximum increase multiplier (default: 2.0 = 200% increase max)
	MaxIncreaseMultiplier float64
	// Maximum decrease multiplier (default: 0.5 = 50% decrease max)
	MaxDecreaseMultiplier float64
}

// Returns default recommendation configuration
func DefaultRecommendationConfig() RecommendationConfig {
	return RecommendationConfig{
		Mode:                        ModeBurstable,    // Default: optimize for efficiency
		TargetRequestUtilizationMin: 0.50,             // 50% min (aligns with analyzer)
		TargetRequestUtilizationMax: 0.70,             // 70% max (aligns with analyzer)
		RequestSafetyMargin:         1.2,              // 20% headroom
		LimitSafetyMargin:           1.3,              // 30% headroom
		MinCPURequest:               0.01,             // 10m minimum
		MinMemoryRequest:            64 * 1024 * 1024, // 64Mi minimum
		MinConfidenceThreshold:      0.5,              // 50% minimum confidence
		MaxIncreaseMultiplier:       2.0,              // Max 2x increase
		MaxDecreaseMultiplier:       0.5,              // Max 50% decrease
	}
}

// Validates recommendation configuration
func (config RecommendationConfig) Validate() error {
	if config.Mode != ModeBurstable && config.Mode != ModeGuaranteed {
		return fmt.Errorf("Mode must be either %s or %s, got: %v", ModeBurstable, ModeGuaranteed, config.Mode)
	}
	if config.Mode == ModeBurstable {
		// Target utilization only required for burstable mode
		if config.TargetRequestUtilizationMin <= 0 {
			return fmt.Errorf("TargetRequestUtilizationMin must be positive, got: %v", config.TargetRequestUtilizationMin)
		}
		if config.TargetRequestUtilizationMax <= 0 {
			return fmt.Errorf("TargetRequestUtilizationMax must be positive, got: %v", config.TargetRequestUtilizationMax)
		}
		if config.TargetRequestUtilizationMin >= config.TargetRequestUtilizationMax {
			return fmt.Errorf("TargetRequestUtilizationMin must be less than TargetRequestUtilizationMax, got: min=%v max=%v", config.TargetRequestUtilizationMin, config.TargetRequestUtilizationMax)
		}
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
	if config.MaxIncreaseMultiplier <= 1.0 {
		return fmt.Errorf("MaxIncreaseMultiplier must be greater than 1.0, got: %v", config.MaxIncreaseMultiplier)
	}
	if config.MaxDecreaseMultiplier <= 0 || config.MaxDecreaseMultiplier >= 1.0 {
		return fmt.Errorf("MaxDecreaseMultiplier must be between 0 and 1.0, got: %v", config.MaxDecreaseMultiplier)
	}
	return nil
}

// Calculates CPU request recommendation based on utilization analysis
func CalculateCPURequestRecommendation(
	currentRequest *float64,
	utilization analyzer.CPUUtilization,
	provisioning analyzer.CPUProvisioning,
	config RecommendationConfig,
) (*float64, RecommendationReason, error) {
	// Adjust confidence threshold based on efficiency
	effectiveThreshold := config.MinConfidenceThreshold
	if provisioning.Efficiency < 0.3 {
		// Low efficiency = be more aggressive (lower threshold)
		effectiveThreshold = effectiveThreshold * 0.8
	}

	// If we don't have enough confidence, don't recommend
	if provisioning.Confidence < effectiveThreshold {
		return nil, ReasonNoData, nil
	}

	// Choose percentile based on anomalies and variance
	// If many anomalies or high variance, use P99 for more conservative recommendation
	percentile := utilization.Stats.Percentile.P95
	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	// Use P99 if: many anomalies, high variance, or P99 is much higher than P95
	if utilization.Anomalies.AnomalyCount > 5 ||
		cv > 0.5 ||
		utilization.Stats.Percentile.P99 > utilization.Stats.Percentile.P95*1.5 {
		percentile = utilization.Stats.Percentile.P99
	}

	// Calculate recommended request to achieve target utilization
	// recommended = percentileUsage / targetUtilization * safetyMargin
	safetyMargin := config.RequestSafetyMargin

	// Adjust safety margin based on trend
	if utilization.Trend.Direction == analyzer.DirectionIncreasing && utilization.Trend.Strength > 0.5 {
		// Increasing trend with strong signal = add extra headroom
		safetyMargin *= 1.1
	} else if utilization.Trend.Direction == analyzer.DirectionDecreasing && utilization.Trend.Strength > 0.5 {
		// Decreasing trend = can be slightly less conservative
		safetyMargin *= 0.95
	}

	// Adjust safety margin based on variance (coefficient of variation)
	if cv > 0.5 {
		// High variance = more conservative
		safetyMargin *= 1.15
	} else if cv < 0.2 {
		// Low variance = can be more precise
		safetyMargin *= 0.98
	}

	// Adjust safety margin based on anomalies
	if utilization.Anomalies.AnomalyCount > 10 {
		// Many anomalies = more conservative
		safetyMargin *= 1.1
	}

	// Calculate recommended request based on mode
	var recommendedCores float64
	if config.Mode == ModeGuaranteed {
		// Guaranteed mode: request = peak usage * safety margin (no throttling risk)
		peakUsage := utilization.Stats.Max
		recommendedCores = peakUsage * safetyMargin
	} else {
		// Burstable mode: optimize for efficiency using target utilization
		// Use midpoint of target range for calculation
		targetUtilization := (config.TargetRequestUtilizationMin + config.TargetRequestUtilizationMax) / 2.0
		recommendedCores = (percentile / targetUtilization) * safetyMargin
	}

	// Validation: ensure recommended is at least greater than mean (sanity check)
	if utilization.Stats.Mean > 0 && recommendedCores < utilization.Stats.Mean {
		recommendedCores = utilization.Stats.Mean * 1.1 // At least 10% above mean
	}

	// Apply minimum
	if recommendedCores < config.MinCPURequest {
		recommendedCores = config.MinCPURequest
	}

	// Apply maximum increase/decrease bounds to prevent excessive changes
	if currentRequest != nil && *currentRequest > 0 {
		current := *currentRequest

		// Prevent excessive increases
		maxAllowed := current * config.MaxIncreaseMultiplier
		if recommendedCores > maxAllowed {
			recommendedCores = maxAllowed
		}

		// Prevent excessive decreases
		minAllowed := current * config.MaxDecreaseMultiplier
		if recommendedCores < minAllowed {
			recommendedCores = minAllowed
		}
	}

	// Round to reasonable precision (3 decimal places for CPU)
	recommendedCores = math.Round(recommendedCores*1000) / 1000

	// Determine reason
	var reason RecommendationReason
	if currentRequest == nil {
		reason = ReasonUnderProvisioned // No request set, recommend one
	} else if provisioning.IsOverProvisioned {
		reason = ReasonOverProvisioned
	} else if provisioning.IsUnderProvisioned {
		reason = ReasonUnderProvisioned
	} else {
		reason = ReasonOptimal
	}

	// Only recommend if there's a meaningful change (at least 10% difference)
	if currentRequest != nil {
		current := *currentRequest
		// If current is 0, always recommend (no request set)
		if current > 0 {
			diff := math.Abs(recommendedCores - current)
			if diff < current*0.1 {
				// Less than 10% difference, consider it optimal
				return nil, ReasonOptimal, nil
			}
		}
	}

	return &recommendedCores, reason, nil
}

// Calculates CPU limit recommendation based on utilization analysis
func CalculateCPULimitRecommendation(
	currentLimit *float64,
	utilization analyzer.CPUUtilization,
	provisioning analyzer.CPUProvisioning,
	config RecommendationConfig,
	recommendedRequest *float64,
) (*float64, RecommendationReason, error) {
	// Adjust confidence threshold based on efficiency
	effectiveThreshold := config.MinConfidenceThreshold
	if provisioning.Efficiency < 0.3 {
		effectiveThreshold = effectiveThreshold * 0.8
	}

	// If we don't have enough confidence, don't recommend
	if provisioning.Confidence < effectiveThreshold {
		return nil, ReasonNoData, nil
	}

	// Use peak (max) for limit recommendation with safety margin
	peakUsage := utilization.Stats.Max

	// Adjust safety margin based on trend
	safetyMargin := config.LimitSafetyMargin
	if utilization.Trend.Direction == analyzer.DirectionIncreasing && utilization.Trend.Strength > 0.5 {
		// Increasing trend = add extra headroom
		safetyMargin *= 1.1
	}

	// Adjust safety margin based on variance
	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}
	if cv > 0.5 {
		safetyMargin *= 1.15
	}

	// Adjust safety margin based on anomalies
	if utilization.Anomalies.AnomalyCount > 10 {
		safetyMargin *= 1.1
	}

	// Calculate recommended limit: peak * safetyMargin
	recommendedCores := peakUsage * safetyMargin

	// Apply minimum
	if recommendedCores < config.MinCPURequest {
		recommendedCores = config.MinCPURequest
	}

	// Ensure limit is at least equal to recommended request
	if recommendedRequest != nil && recommendedCores < *recommendedRequest {
		recommendedCores = *recommendedRequest
	}

	// Apply maximum increase/decrease bounds to prevent excessive changes
	if currentLimit != nil && *currentLimit > 0 {
		current := *currentLimit

		// Prevent excessive increases
		maxAllowed := current * config.MaxIncreaseMultiplier
		if recommendedCores > maxAllowed {
			recommendedCores = maxAllowed
		}

		// Prevent excessive decreases
		minAllowed := current * config.MaxDecreaseMultiplier
		if recommendedCores < minAllowed {
			recommendedCores = minAllowed
		}
	}

	// Round to reasonable precision
	recommendedCores = math.Round(recommendedCores*1000) / 1000

	// Determine reason
	var reason RecommendationReason
	if currentLimit == nil {
		reason = ReasonUnderProvisioned // No limit set, recommend one
	} else if provisioning.LimitUtilization > (1.0 - 0.2) {
		// Peak too close to limit (within 20%)
		reason = ReasonUnderProvisioned
	} else {
		reason = ReasonOptimal
	}

	// Only recommend if there's a meaningful change (at least 10% difference)
	if currentLimit != nil {
		current := *currentLimit
		// If current is 0, always recommend (no limit set)
		if current > 0 {
			diff := math.Abs(recommendedCores - current)
			if diff < current*0.1 {
				// Less than 10% difference, consider it optimal
				return nil, ReasonOptimal, nil
			}
		}
	}

	return &recommendedCores, reason, nil
}

// Calculates memory request recommendation based on utilization analysis
func CalculateMemoryRequestRecommendation(
	currentRequest *float64,
	utilization analyzer.MemoryUtilization,
	provisioning analyzer.MemoryProvisioning,
	config RecommendationConfig,
) (*float64, RecommendationReason, error) {
	// Adjust confidence threshold based on efficiency
	effectiveThreshold := config.MinConfidenceThreshold
	if provisioning.Efficiency < 0.3 {
		effectiveThreshold = effectiveThreshold * 0.8
	}

	// If we don't have enough confidence, don't recommend
	if provisioning.Confidence < effectiveThreshold {
		return nil, ReasonNoData, nil
	}

	// Choose percentile based on anomalies and variance
	percentile := utilization.Stats.Percentile.P95
	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}

	// Use P99 if: many anomalies, high variance, or P99 is much higher than P95
	if utilization.Anomalies.AnomalyCount > 5 ||
		cv > 0.5 ||
		utilization.Stats.Percentile.P99 > utilization.Stats.Percentile.P95*1.5 {
		percentile = utilization.Stats.Percentile.P99
	}

	// Calculate recommended request to achieve target utilization
	safetyMargin := config.RequestSafetyMargin

	// Adjust safety margin based on trend
	if utilization.Trend.Direction == analyzer.DirectionIncreasing && utilization.Trend.Strength > 0.5 {
		safetyMargin *= 1.1
	} else if utilization.Trend.Direction == analyzer.DirectionDecreasing && utilization.Trend.Strength > 0.5 {
		safetyMargin *= 0.95
	}

	// Adjust safety margin based on variance
	if cv > 0.5 {
		safetyMargin *= 1.15
	} else if cv < 0.2 {
		safetyMargin *= 0.98
	}

	// Adjust safety margin based on anomalies
	if utilization.Anomalies.AnomalyCount > 10 {
		safetyMargin *= 1.1
	}

	// Calculate recommended request based on mode
	var recommendedBytes float64
	if config.Mode == ModeGuaranteed {
		// Guaranteed mode: request = peak usage * safety margin (no throttling risk)
		peakUsage := utilization.Stats.Max
		recommendedBytes = peakUsage * safetyMargin
	} else {
		// Burstable mode: optimize for efficiency using target utilization
		// Use midpoint of target range for calculation
		targetUtilization := (config.TargetRequestUtilizationMin + config.TargetRequestUtilizationMax) / 2.0
		recommendedBytes = (percentile / targetUtilization) * safetyMargin
	}

	// Validation: ensure recommended is at least greater than mean
	if utilization.Stats.Mean > 0 && recommendedBytes < utilization.Stats.Mean {
		recommendedBytes = utilization.Stats.Mean * 1.1
	}

	// Apply minimum
	if recommendedBytes < float64(config.MinMemoryRequest) {
		recommendedBytes = float64(config.MinMemoryRequest)
	}

	// Apply maximum increase/decrease bounds to prevent excessive changes
	if currentRequest != nil && *currentRequest > 0 {
		current := *currentRequest

		// Prevent excessive increases
		maxAllowed := current * config.MaxIncreaseMultiplier
		if recommendedBytes > maxAllowed {
			recommendedBytes = maxAllowed
		}

		// Prevent excessive decreases
		minAllowed := current * config.MaxDecreaseMultiplier
		if recommendedBytes < minAllowed {
			recommendedBytes = minAllowed
		}
	}

	// Round to nearest byte
	recommendedBytes = math.Round(recommendedBytes)

	// Determine reason
	var reason RecommendationReason
	if currentRequest == nil {
		reason = ReasonUnderProvisioned // No request set, recommend one
	} else if provisioning.IsOverProvisioned {
		reason = ReasonOverProvisioned
	} else if provisioning.IsUnderProvisioned {
		reason = ReasonUnderProvisioned
	} else {
		reason = ReasonOptimal
	}

	// Only recommend if there's a meaningful change (at least 10% difference)
	if currentRequest != nil {
		current := *currentRequest
		// If current is 0, always recommend (no request set)
		if current > 0 {
			diff := math.Abs(recommendedBytes - current)
			if diff < current*0.1 {
				// Less than 10% difference, consider it optimal
				return nil, ReasonOptimal, nil
			}
		}
	}

	return &recommendedBytes, reason, nil
}

// Calculates memory limit recommendation based on utilization analysis
func CalculateMemoryLimitRecommendation(
	currentLimit *float64,
	utilization analyzer.MemoryUtilization,
	provisioning analyzer.MemoryProvisioning,
	config RecommendationConfig,
	recommendedRequest *float64,
) (*float64, RecommendationReason, error) {
	// Adjust confidence threshold based on efficiency
	effectiveThreshold := config.MinConfidenceThreshold
	if provisioning.Efficiency < 0.3 {
		effectiveThreshold = effectiveThreshold * 0.8
	}

	// If we don't have enough confidence, don't recommend
	if provisioning.Confidence < effectiveThreshold {
		return nil, ReasonNoData, nil
	}

	// Use peak (max) for limit recommendation with safety margin
	peakUsage := utilization.Stats.Max

	// Adjust safety margin based on trend
	safetyMargin := config.LimitSafetyMargin
	if utilization.Trend.Direction == analyzer.DirectionIncreasing && utilization.Trend.Strength > 0.5 {
		safetyMargin *= 1.1
	}

	// Adjust safety margin based on variance
	cv := 0.0
	if utilization.Stats.Mean > 0 {
		cv = utilization.Stats.StdDev / utilization.Stats.Mean
	}
	if cv > 0.5 {
		safetyMargin *= 1.15
	}

	// Adjust safety margin based on anomalies
	if utilization.Anomalies.AnomalyCount > 10 {
		safetyMargin *= 1.1
	}

	// Calculate recommended limit: peak * safetyMargin
	recommendedBytes := peakUsage * safetyMargin

	// Apply minimum
	if recommendedBytes < float64(config.MinMemoryRequest) {
		recommendedBytes = float64(config.MinMemoryRequest)
	}

	// Ensure limit is at least equal to recommended request
	if recommendedRequest != nil && recommendedBytes < *recommendedRequest {
		recommendedBytes = *recommendedRequest
	}

	// Apply maximum increase/decrease bounds to prevent excessive changes
	if currentLimit != nil && *currentLimit > 0 {
		current := *currentLimit

		// Prevent excessive increases
		maxAllowed := current * config.MaxIncreaseMultiplier
		if recommendedBytes > maxAllowed {
			recommendedBytes = maxAllowed
		}

		// Prevent excessive decreases
		minAllowed := current * config.MaxDecreaseMultiplier
		if recommendedBytes < minAllowed {
			recommendedBytes = minAllowed
		}
	}

	// Round to nearest byte
	recommendedBytes = math.Round(recommendedBytes)

	// Determine reason
	var reason RecommendationReason
	if currentLimit == nil {
		reason = ReasonUnderProvisioned // No limit set, recommend one
	} else if provisioning.LimitUtilization > (1.0 - 0.2) {
		// Peak too close to limit (within 20%)
		reason = ReasonUnderProvisioned
	} else {
		reason = ReasonOptimal
	}

	// Only recommend if there's a meaningful change (at least 10% difference)
	if currentLimit != nil {
		current := *currentLimit
		// If current is 0, always recommend (no limit set)
		if current > 0 {
			diff := math.Abs(recommendedBytes - current)
			if diff < current*0.1 {
				// Less than 10% difference, consider it optimal
				return nil, ReasonOptimal, nil
			}
		}
	}

	return &recommendedBytes, reason, nil
}

// Calculates overall confidence score from CPU and memory provisioning
func CalculateOverallConfidence(
	cpuProvisioning analyzer.CPUProvisioning,
	memoryProvisioning analyzer.MemoryProvisioning,
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

	// If no limit recommendation, but we have a request, we might need to recommend a limit
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
