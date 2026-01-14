package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/thread_koder/mochi/internal/database"
)

// Represents compute resource recommendations for a workload
type Recommendation struct {
	WorkloadType       string                    `json:"workload_type"` // Deployment, StatefulSet, DaemonSet, Pod
	WorkloadName       string                    `json:"workload_name"`
	Namespace          string                    `json:"namespace"`
	RecommendationMode string                    `json:"recommendation_mode"` // "cost_optimized", "burstable", or "guaranteed"
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
		config,
		memoryRequestRecValue,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate memory limit recommendation: %w", err)
	}

	// Ensure limits are >= requests
	cpuLimitRecValue = ensureLimitGreaterThanRequestValue(
		cpuLimitRecValue, cpuRequestRecValue, specs.CPULimit, specs.CPURequest,
	)
	memoryLimitRecValue = ensureLimitGreaterThanRequestValue(
		memoryLimitRecValue, memoryRequestRecValue, specs.MemoryLimit, specs.MemoryRequest,
	)

	// Calculate overall confidence
	overallConfidence := CalculateOverallConfidence(
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

			// For each unique container name, take the recommendation with higher confidence
			existingRec, exists := containerRecsMap[container.Name]
			if !exists {
				// First instance of this container name
				containerRecsMap[container.Name] = rec
			} else {
				// Replace if new recommendation has higher confidence
				if rec.ConfidenceScore > existingRec.ConfidenceScore {
					containerRecsMap[container.Name] = rec
				}
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
		RecommendationMode: string(config.Mode),
		Recommendations:    recommendations,
		AnalysisTimeRange:  analysisOpts.TimeRange.String(),
	}

	return result, nil
}

// Calculates change percentage between current and recommended values
// Returns 100.0 if current is nil (new resource)
// Returns nil if recommended is nil
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
		RecommendationMode: rec.RecommendationMode,
		Recommendations:    recommendationsJSON,
		Status:             "pending",
		AnalysisTimeRange:  rec.AnalysisTimeRange,
		CreatedAt:          now,
		UpdatedAt:          now,
		GeneratedAt:        now,
	}, nil
}
