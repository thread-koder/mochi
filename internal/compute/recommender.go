package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/thread_koder/mochi/internal/database"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Represents compute resource recommendations for a workload
type Recommendation struct {
	WorkloadType       string                    `json:"workload_type"` // Deployment, StatefulSet, DaemonSet, Pod
	WorkloadName       string                    `json:"workload_name"`
	Namespace          string                    `json:"namespace"`
	RecommendationMode RecommendationMode        `json:"recommendation_mode"` // "cost_optimized", "burstable", or "guaranteed"
	Recommendations    []ContainerRecommendation `json:"recommendations"`
	AnalysisTimeRange  string                    `json:"analysis_time_range"`
}

// Represents a container recommendation
type ContainerRecommendation struct {
	ContainerName   string               `json:"container_name"`
	CPU             CPURecommendation    `json:"cpu"`
	Memory          MemoryRecommendation `json:"memory"`
	ConfidenceScore float64              `json:"confidence_score"`
}

// Represents CPU resource recommendations
type CPURecommendation struct {
	CurrentRequest       *string  `json:"current_request"`
	RecommendedRequest   *string  `json:"recommended_request"`
	RequestChangePercent *float64 `json:"request_change_percent"` // Rounded to 1 decimal
	CurrentLimit         *string  `json:"current_limit"`
	RecommendedLimit     *string  `json:"recommended_limit"`
	LimitChangePercent   *float64 `json:"limit_change_percent"` // Rounded to 1 decimal
}

// Represents memory resource recommendations
type MemoryRecommendation struct {
	CurrentRequest       *string  `json:"current_request"`
	RecommendedRequest   *string  `json:"recommended_request"`
	RequestChangePercent *float64 `json:"request_change_percent"` // Rounded to 1 decimal
	CurrentLimit         *string  `json:"current_limit"`
	RecommendedLimit     *string  `json:"recommended_limit"`
	LimitChangePercent   *float64 `json:"limit_change_percent"` // Rounded to 1 decimal
}

// Generates a resource recommendation for a container based on analysis results
func GenerateContainerRecommendation(
	ctx context.Context,
	container *database.Container,
	containerAnalysis ContainerAnalysis,
	config RecommendationConfig,
) (*ContainerRecommendation, error) {
	// Validate inputs
	if container == nil {
		return nil, fmt.Errorf("container cannot be nil")
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid recommendation config: %w", err)
	}

	// Parse current resource specs
	specs, err := ParseContainerSpecs(container)
	if err != nil {
		return nil, fmt.Errorf("failed to parse container specs: %w", err)
	}

	// Calculate CPU request recommendation
	cpuRequestRecValue, err := CalculateCPURequestRecommendation(
		specs.CPURequest,
		containerAnalysis.Utilization.CPU,
		containerAnalysis.Provisioning.CPU,
		containerAnalysis.Stability,
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate CPU request recommendation: %w", err)
	}

	// Calculate CPU limit recommendation
	cpuLimitRecValue, err := CalculateCPULimitRecommendation(
		specs.CPULimit,
		containerAnalysis.Utilization.CPU,
		containerAnalysis.Provisioning.CPU,
		containerAnalysis.Stability,
		config,
		cpuRequestRecValue,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate CPU limit recommendation: %w", err)
	}

	// Calculate memory request recommendation
	memoryRequestRecValue, err := CalculateMemoryRequestRecommendation(
		specs.MemoryRequest,
		containerAnalysis.Utilization.Memory,
		containerAnalysis.Provisioning.Memory,
		containerAnalysis.Stability,
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate memory request recommendation: %w", err)
	}

	// Calculate memory limit recommendation
	memoryLimitRecValue, err := CalculateMemoryLimitRecommendation(
		specs.MemoryLimit,
		containerAnalysis.Utilization.Memory,
		containerAnalysis.Provisioning.Memory,
		containerAnalysis.Stability,
		config,
		memoryRequestRecValue,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate memory limit recommendation: %w", err)
	}

	cpuRequestRecValue, cpuLimitRecValue = finalizeResourceRecommendations(
		cpuRequestRecValue, cpuLimitRecValue, specs.CPURequest, specs.CPULimit, config.Mode,
	)

	memoryRequestRecValue, memoryLimitRecValue = finalizeResourceRecommendations(
		memoryRequestRecValue, memoryLimitRecValue, specs.MemoryRequest, specs.MemoryLimit, config.Mode,
	)

	// Calculate overall confidence
	overallConfidence := calculateOverallConfidence(
		containerAnalysis.Provisioning.CPU,
		containerAnalysis.Provisioning.Memory,
	)

	// Only create recommendation if we have at least one recommendation and sufficient confidence
	hasRecommendation := cpuRequestRecValue != nil || cpuLimitRecValue != nil || memoryRequestRecValue != nil || memoryLimitRecValue != nil
	if !hasRecommendation || overallConfidence < config.MinConfidenceThreshold {
		return nil, nil
	}

	// Calculate change percentages
	cpuRequestChangePercent := calculateChangePercent(specs.CPURequest, cpuRequestRecValue)
	cpuLimitChangePercent := calculateChangePercent(specs.CPULimit, cpuLimitRecValue)
	memoryRequestChangePercent := calculateChangePercent(specs.MemoryRequest, memoryRequestRecValue)
	memoryLimitChangePercent := calculateChangePercent(specs.MemoryLimit, memoryLimitRecValue)

	// Format recommendations to Kubernetes resource quantity strings
	var cpuRequestRec, cpuLimitRec, memoryRequestRec, memoryLimitRec *string
	if cpuRequestRecValue != nil {
		formatted := formatCPUQuantity(*cpuRequestRecValue)
		cpuRequestRec = &formatted
	}
	if cpuLimitRecValue != nil {
		formatted := formatCPUQuantity(*cpuLimitRecValue)
		cpuLimitRec = &formatted
	}
	if memoryRequestRecValue != nil {
		formatted := formatMemoryQuantity(int64(*memoryRequestRecValue))
		memoryRequestRec = &formatted
	}
	if memoryLimitRecValue != nil {
		formatted := formatMemoryQuantity(int64(*memoryLimitRecValue))
		memoryLimitRec = &formatted
	}

	// Build recommendation response
	recommendation := &ContainerRecommendation{
		ContainerName: container.Name,
		CPU: CPURecommendation{
			CurrentRequest:       container.CPURequest,
			RecommendedRequest:   cpuRequestRec,
			RequestChangePercent: cpuRequestChangePercent,
			CurrentLimit:         container.CPULimit,
			RecommendedLimit:     cpuLimitRec,
			LimitChangePercent:   cpuLimitChangePercent,
		},
		Memory: MemoryRecommendation{
			CurrentRequest:       container.MemoryRequest,
			RecommendedRequest:   memoryRequestRec,
			RequestChangePercent: memoryRequestChangePercent,
			CurrentLimit:         container.MemoryLimit,
			RecommendedLimit:     memoryLimitRec,
			LimitChangePercent:   memoryLimitChangePercent,
		},
		ConfidenceScore: overallConfidence,
	}

	return recommendation, nil
}

// Generates recommendations for a workload
// For workloads with multiple replicas, analyzes all instances and takes the recommendation with higher confidence per unique container name
func GenerateWorkloadRecommendations(
	ctx context.Context,
	workloadType string,
	workloadName string,
	namespace string,
	pods []*database.Pod,
	config RecommendationConfig,
	analysisOpts AnalysisOptions,
) (Recommendation, error) {
	// Validate inputs
	if len(pods) == 0 {
		return Recommendation{}, fmt.Errorf("no pods provided for workload %s/%s", namespace, workloadName)
	}

	if err := config.Validate(); err != nil {
		return Recommendation{}, fmt.Errorf("invalid recommendation config: %w", err)
	}

	if err := analysisOpts.Validate(); err != nil {
		return Recommendation{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	// Key: container name, Value: ContainerRecommendation
	containerRecsMap := make(map[string]*ContainerRecommendation)

	// Analyze each pod and its containers
	for _, pod := range pods {
		// Get containers for this pod
		containers, err := database.GetContainersByPodUID(ctx, pod.UID)
		if err != nil {
			return Recommendation{}, fmt.Errorf("failed to fetch containers for pod %s: %w", pod.Name, err)
		}

		// Analyze each container
		for _, container := range containers {
			// Analyze container
			containerAnalysis, err := AnalyzeContainer(ctx, container, analysisOpts)
			if err != nil {
				return Recommendation{}, fmt.Errorf("failed to analyze container %s: %w", container.Name, err)
			}

			// Generate recommendation for this container instance
			rec, err := GenerateContainerRecommendation(ctx, container, containerAnalysis, config)
			if err != nil {
				return Recommendation{}, fmt.Errorf("failed to generate container recommendation for %s: %w", container.Name, err)
			}

			// Skip if no recommendation generated
			if rec == nil {
				continue
			}

			// For each unique container name, take the Maximum recommended values across all replicas
			existingRec, exists := containerRecsMap[container.Name]
			if !exists {
				// First instance of this container name
				containerRecsMap[container.Name] = rec
			} else {
				// Max CPU Request
				updateMaxQuantity(&existingRec.CPU.RecommendedRequest, rec.CPU.RecommendedRequest)
				// Max CPU Limit
				updateMaxQuantity(&existingRec.CPU.RecommendedLimit, rec.CPU.RecommendedLimit)
				// Max Memory Request
				updateMaxQuantity(&existingRec.Memory.RecommendedRequest, rec.Memory.RecommendedRequest)
				// Max Memory Limit
				updateMaxQuantity(&existingRec.Memory.RecommendedLimit, rec.Memory.RecommendedLimit)

				// Keep highest confidence score
				if rec.ConfidenceScore > existingRec.ConfidenceScore {
					existingRec.ConfidenceScore = rec.ConfidenceScore
				}

				// Recalculate change percentages based on new max values
				existingRec.CPU.RequestChangePercent = calculateChangePercentFromStrings(existingRec.CPU.CurrentRequest, existingRec.CPU.RecommendedRequest)
				existingRec.CPU.LimitChangePercent = calculateChangePercentFromStrings(existingRec.CPU.CurrentLimit, existingRec.CPU.RecommendedLimit)
				existingRec.Memory.RequestChangePercent = calculateChangePercentFromStrings(existingRec.Memory.CurrentRequest, existingRec.Memory.RecommendedRequest)
				existingRec.Memory.LimitChangePercent = calculateChangePercentFromStrings(existingRec.Memory.CurrentLimit, existingRec.Memory.RecommendedLimit)
			}
		}
	}

	// Convert map to slice
	recommendations := make([]ContainerRecommendation, 0, len(containerRecsMap))
	for _, rec := range containerRecsMap {
		recommendations = append(recommendations, *rec)
	}

	result := Recommendation{
		WorkloadType:       workloadType,
		WorkloadName:       workloadName,
		Namespace:          namespace,
		RecommendationMode: config.Mode,
		Recommendations:    recommendations,
		AnalysisTimeRange:  analysisOpts.TimeRange.String(),
	}

	return result, nil
}

// Converts a compute Recommendation to a database ComputeRecommendation
func ComputeRecommendationToDB(rec Recommendation) (*database.ComputeRecommendation, error) {
	recommendationsJSON, err := json.Marshal(rec.Recommendations)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal recommendations: %w", err)
	}

	now := time.Now()
	return &database.ComputeRecommendation{
		WorkloadType:       rec.WorkloadType,
		WorkloadName:       rec.WorkloadName,
		Namespace:          rec.Namespace,
		RecommendationMode: string(rec.RecommendationMode),
		Recommendations:    recommendationsJSON,
		Status:             "pending",
		AnalysisTimeRange:  rec.AnalysisTimeRange,
		CreatedAt:          now,
		UpdatedAt:          now,
		GeneratedAt:        now,
	}, nil
}

// Updates a target quantity string if the source is larger
func updateMaxQuantity(target **string, source *string) {
	if source == nil {
		return
	}
	if *target == nil {
		*target = source
		return
	}

	q1, err1 := resource.ParseQuantity(**target)
	q2, err2 := resource.ParseQuantity(*source)
	if err1 != nil || err2 != nil {
		return
	}

	if q2.Cmp(q1) > 0 {
		*target = source
	}
}

// Calculates change percent from quantity strings
func calculateChangePercentFromStrings(currentStr, recommendedStr *string) *float64 {
	if recommendedStr == nil {
		return nil
	}

	var curVal float64
	if currentStr != nil && *currentStr != "" {
		q, err := resource.ParseQuantity(*currentStr)
		if err == nil {
			curVal = q.AsFloat64Slow()
		}
	}

	qRec, err := resource.ParseQuantity(*recommendedStr)
	if err != nil {
		return nil
	}
	recVal := qRec.AsFloat64Slow()

	return calculateChangePercent(&curVal, &recVal)
}

// Calculates change percentage between current and recommended values
// Returns 100.0 if current is nil (new resource)
func calculateChangePercent(current, recommended *float64) *float64 {
	if recommended == nil {
		return nil
	}

	if current == nil || *current == 0 {
		// New resource or zero current, return 100.0%
		val := 100.0
		return &val
	}

	// Calculate percentage
	changePercent := ((*recommended - *current) / *current) * 100.0

	// Round to 1 decimal place
	rounded := math.Round(changePercent*10) / 10

	return &rounded
}
