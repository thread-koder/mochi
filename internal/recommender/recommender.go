package recommender

import (
	"context"
	"fmt"
	"time"

	"github.com/thread_koder/mochi/internal/analyzer"
	"github.com/thread_koder/mochi/internal/database"
)

// Generates a resource recommendation for a container based on analysis results
func GenerateContainerRecommendation(
	ctx context.Context,
	container *database.Container,
	containerAnalysis analyzer.ContainerAnalysis,
	config RecommendationConfig,
) (*database.ContainerRecommendation, error) {
	// Validate inputs
	if container == nil {
		return nil, fmt.Errorf("container cannot be nil")
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid recommendation config: %w", err)
	}

	// Parse current resource specs using the analyzer package's function
	specs, err := analyzer.ParseContainerSpecs(container)
	if err != nil {
		return nil, fmt.Errorf("failed to parse container specs: %w", err)
	}

	// Calculate CPU request recommendation
	cpuRequestRecValue, _, err := CalculateCPURequestRecommendation(
		specs.CPURequest,
		containerAnalysis.Utilization.CPU,
		containerAnalysis.Provisioning.CPU,
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate CPU request recommendation: %w", err)
	}

	// Calculate CPU limit recommendation
	cpuLimitRecValue, _, err := CalculateCPULimitRecommendation(
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
	memoryRequestRecValue, _, err := CalculateMemoryRequestRecommendation(
		specs.MemoryRequest,
		containerAnalysis.Utilization.Memory,
		containerAnalysis.Provisioning.Memory,
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate memory request recommendation: %w", err)
	}

	// Calculate memory limit recommendation
	memoryLimitRecValue, _, err := CalculateMemoryLimitRecommendation(
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

	// Calculate overall confidence
	overallConfidence := CalculateOverallConfidence(
		containerAnalysis.Provisioning.CPU,
		containerAnalysis.Provisioning.Memory,
	)

	// Only create recommendation if we have at least one recommendation and sufficient confidence
	hasRecommendation := cpuRequestRec != nil || cpuLimitRec != nil || memoryRequestRec != nil || memoryLimitRec != nil
	if !hasRecommendation || overallConfidence < config.MinConfidenceThreshold {
		return nil, nil
	}

	// Create recommendation record
	recommendation := &database.ContainerRecommendation{
		ContainerID:              container.ID,
		PodUID:                   container.PodUID,
		ContainerName:            container.Name,
		Namespace:                container.Namespace,
		CurrentCPURequest:        container.CPURequest,
		CurrentCPULimit:          container.CPULimit,
		CurrentMemoryRequest:     container.MemoryRequest,
		CurrentMemoryLimit:       container.MemoryLimit,
		RecommendedCPURequest:    cpuRequestRec,
		RecommendedCPULimit:      cpuLimitRec,
		RecommendedMemoryRequest: memoryRequestRec,
		RecommendedMemoryLimit:   memoryLimitRec,
		RecommendationMode:       string(config.Mode),
		ConfidenceScore:          overallConfidence,
		Status:                   "pending",
		CreatedAt:                time.Now(),
	}

	return recommendation, nil
}

// Generates and stores a recommendation for a container
func GenerateAndStoreRecommendation(
	ctx context.Context,
	container *database.Container,
	containerAnalysis analyzer.ContainerAnalysis,
	config RecommendationConfig,
) (*database.ContainerRecommendation, error) {
	// Generate recommendation
	recommendation, err := GenerateContainerRecommendation(ctx, container, containerAnalysis, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate recommendation: %w", err)
	}

	// If no recommendation was generated (e.g., optimal or low confidence), return nil
	if recommendation == nil {
		return nil, nil
	}

	// Store recommendation in database
	if err := database.UpsertContainerRecommendation(ctx, recommendation); err != nil {
		return nil, fmt.Errorf("failed to store recommendation: %w", err)
	}

	return recommendation, nil
}
