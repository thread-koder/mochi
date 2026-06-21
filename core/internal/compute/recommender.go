package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/thread_koder/mochi/core/internal/database"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Recommendation is a workload-level response: mode, per-container suggested quantities as Kubernetes
// quantity strings, and the analysis window used.
type Recommendation struct {
	WorkloadType       string                    `json:"workload_type"`
	WorkloadName       string                    `json:"workload_name"`
	Namespace          string                    `json:"namespace"`
	RecommendationMode RecommendationMode        `json:"recommendation_mode"`
	Recommendations    []ContainerRecommendation `json:"recommendations"`
	AnalysisTimeRange  string                    `json:"analysis_time_range"`
}

// ContainerRecommendation is one container’s suggested CPU and memory requests/limits plus a confidence score.
type ContainerRecommendation struct {
	ContainerName string               `json:"container_name"`
	CPU           CPURecommendation    `json:"cpu"`
	Memory        MemoryRecommendation `json:"memory"`
	Confidence    float64              `json:"confidence"`
}

// CPURecommendation holds current and recommended CPU request/limit strings, percent deltas (one decimal),
// and per-resource confidence for pod merge.
type CPURecommendation struct {
	CurrentRequest       *string  `json:"current_request"`
	RecommendedRequest   *string  `json:"recommended_request"`
	RequestChangePercent *float64 `json:"request_change_percent"`
	CurrentLimit         *string  `json:"current_limit"`
	RecommendedLimit     *string  `json:"recommended_limit"`
	LimitChangePercent   *float64 `json:"limit_change_percent"`
	Confidence           float64  `json:"-"`
}

// MemoryRecommendation holds current and recommended memory request/limit strings, percent deltas (one decimal),
// and per-resource confidence for pod merge.
type MemoryRecommendation struct {
	CurrentRequest       *string  `json:"current_request"`
	RecommendedRequest   *string  `json:"recommended_request"`
	RequestChangePercent *float64 `json:"request_change_percent"`
	CurrentLimit         *string  `json:"current_limit"`
	RecommendedLimit     *string  `json:"recommended_limit"`
	LimitChangePercent   *float64 `json:"limit_change_percent"`
	Confidence           float64  `json:"-"`
}

// GenerateContainerRecommendation turns analysis output into a recommendation, applies mode rules
// via finalizeResourceRecommendations, and returns nil when there is nothing to suggest or confidence is
// below MinConfidenceThreshold. When only one of CPU or memory recommends a change, the other resource
// is filled from current specs so the payload stays complete for apply.
func GenerateContainerRecommendation(
	ctx context.Context,
	container *database.Container,
	containerAnalysis ContainerAnalysis,
	config RecommendationConfig,
	analysisWindow time.Duration,
) (*ContainerRecommendation, error) {
	if container == nil {
		return nil, fmt.Errorf("container cannot be nil")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid recommendation config: %w", err)
	}

	specs, err := ParseContainerSpecs(container)
	if err != nil {
		return nil, fmt.Errorf("failed to parse container specs: %w", err)
	}

	cpuRequestRecValue := CalculateCPURequestRecommendation(
		specs.CPURequest,
		containerAnalysis.Utilization.CPU,
		containerAnalysis.Provisioning.CPU,
		containerAnalysis.Stability,
		config,
	)

	cpuLimitRecValue := CalculateCPULimitRecommendation(
		specs.CPULimit,
		containerAnalysis.Utilization.CPU,
		containerAnalysis.Provisioning.CPU,
		containerAnalysis.Stability,
		config,
		cpuRequestRecValue,
		specs.CPURequest,
	)

	memoryRequestRecValue := CalculateMemoryRequestRecommendation(
		specs.MemoryRequest,
		containerAnalysis.Utilization.Memory,
		containerAnalysis.Provisioning.Memory,
		containerAnalysis.Stability,
		config,
		analysisWindow,
	)

	memoryLimitRecValue := CalculateMemoryLimitRecommendation(
		specs.MemoryLimit,
		containerAnalysis.Utilization.Memory,
		containerAnalysis.Provisioning.Memory,
		containerAnalysis.Stability,
		config,
		memoryRequestRecValue,
		specs.MemoryRequest,
		analysisWindow,
	)

	cpuRequestRecValue, cpuLimitRecValue = finalizeResourceRecommendations(
		cpuRequestRecValue, cpuLimitRecValue, specs.CPURequest, specs.CPULimit, config.Mode,
	)

	memoryRequestRecValue, memoryLimitRecValue = finalizeResourceRecommendations(
		memoryRequestRecValue, memoryLimitRecValue, specs.MemoryRequest, specs.MemoryLimit, config.Mode,
	)

	hasCPURecommendation := cpuRequestRecValue != nil || cpuLimitRecValue != nil
	hasMemoryRecommendation := memoryRequestRecValue != nil || memoryLimitRecValue != nil
	if !hasCPURecommendation && !hasMemoryRecommendation {
		return nil, nil
	}

	overallConfidence := calculateOverallConfidence(
		containerAnalysis.Provisioning.CPU.Confidence,
		containerAnalysis.Provisioning.Memory.Confidence,
		hasCPURecommendation,
		hasMemoryRecommendation,
	)

	if overallConfidence < config.MinConfidenceThreshold {
		return nil, nil
	}

	if hasCPURecommendation && !hasMemoryRecommendation {
		if specs.MemoryRequest != nil {
			memoryRequestRecValue = specs.MemoryRequest
		}
		if specs.MemoryLimit != nil {
			memoryLimitRecValue = specs.MemoryLimit
		}
	} else if hasMemoryRecommendation && !hasCPURecommendation {
		if specs.CPURequest != nil {
			cpuRequestRecValue = specs.CPURequest
		}
		if specs.CPULimit != nil {
			cpuLimitRecValue = specs.CPULimit
		}
	}

	cpuRequestChangePercent := calculateChangePercent(specs.CPURequest, cpuRequestRecValue)
	cpuLimitChangePercent := calculateChangePercent(specs.CPULimit, cpuLimitRecValue)
	memoryRequestChangePercent := calculateChangePercent(specs.MemoryRequest, memoryRequestRecValue)
	memoryLimitChangePercent := calculateChangePercent(specs.MemoryLimit, memoryLimitRecValue)

	var cpuRequestRec, cpuLimitRec, memoryRequestRec, memoryLimitRec *string
	if cpuRequestRecValue != nil {
		cpuRequestRec = new(formatCPUQuantity(*cpuRequestRecValue))
	}
	if cpuLimitRecValue != nil {
		cpuLimitRec = new(formatCPUQuantity(*cpuLimitRecValue))
	}
	if memoryRequestRecValue != nil {
		memoryRequestRec = new(formatMemoryQuantity(int64(*memoryRequestRecValue)))
	}
	if memoryLimitRecValue != nil {
		memoryLimitRec = new(formatMemoryQuantity(int64(*memoryLimitRecValue)))
	}

	recommendation := &ContainerRecommendation{
		ContainerName: container.Name,
		CPU: CPURecommendation{
			CurrentRequest:       container.CPURequest,
			RecommendedRequest:   cpuRequestRec,
			RequestChangePercent: cpuRequestChangePercent,
			CurrentLimit:         container.CPULimit,
			RecommendedLimit:     cpuLimitRec,
			LimitChangePercent:   cpuLimitChangePercent,
			Confidence:           containerAnalysis.Provisioning.CPU.Confidence,
		},
		Memory: MemoryRecommendation{
			CurrentRequest:       container.MemoryRequest,
			RecommendedRequest:   memoryRequestRec,
			RequestChangePercent: memoryRequestChangePercent,
			CurrentLimit:         container.MemoryLimit,
			RecommendedLimit:     memoryLimitRec,
			LimitChangePercent:   memoryLimitChangePercent,
			Confidence:           containerAnalysis.Provisioning.Memory.Confidence,
		},
		Confidence: overallConfidence,
	}

	return recommendation, nil
}

// GenerateWorkloadRecommendations analyzes every pod in the workload, generates a container recommendation
// per pod instance, then merges by container name: for each resource, the larger recommended quantity wins
// (so replicas that need more headroom drive the suggestion). Confidence follows the replica that drove
// each resource quantity, change percents are recomputed after merging.
func GenerateWorkloadRecommendations(
	ctx context.Context,
	workloadType string,
	workloadName string,
	namespace string,
	pods []*database.Pod,
	config RecommendationConfig,
	analysisOpts AnalysisOptions,
) (Recommendation, error) {
	if err := config.Validate(); err != nil {
		return Recommendation{}, fmt.Errorf("invalid recommendation config: %w", err)
	}

	if err := analysisOpts.Validate(); err != nil {
		return Recommendation{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	podRecs := make([][]ContainerRecommendation, len(pods))
	g, gctx := errgroup.WithContext(ctx)
	for i, pod := range pods {
		g.Go(func() error {
			recs, err := recommendationsForPod(gctx, pod, config, analysisOpts)
			if err != nil {
				return err
			}
			podRecs[i] = recs
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return Recommendation{}, err
	}

	containerRecsMap := make(map[string]*ContainerRecommendation)
	for _, recs := range podRecs {
		for j := range recs {
			rec := &recs[j]
			existingRec, exists := containerRecsMap[rec.ContainerName]
			if !exists {
				containerRecsMap[rec.ContainerName] = rec
				continue
			}
			mergeContainerRecommendation(existingRec, rec)
		}
	}

	recommendations := make([]ContainerRecommendation, 0, len(containerRecsMap))
	for _, rec := range containerRecsMap {
		recommendations = append(recommendations, *rec)
	}

	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].ContainerName < recommendations[j].ContainerName
	})

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

func recommendationsForPod(
	ctx context.Context,
	pod *database.Pod,
	config RecommendationConfig,
	analysisOpts AnalysisOptions,
) ([]ContainerRecommendation, error) {
	containers, err := database.GetContainersByPodUID(ctx, pod.UID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch containers for pod %s: %w", pod.Name, err)
	}

	containerRecs := make([]*ContainerRecommendation, len(containers))
	g, gctx := errgroup.WithContext(ctx)
	for i, container := range containers {
		g.Go(func() error {
			containerAnalysis, err := AnalyzeContainer(gctx, container, analysisOpts)
			if err != nil {
				return fmt.Errorf("failed to analyze container %s: %w", container.Name, err)
			}

			rec, err := GenerateContainerRecommendation(
				gctx,
				container,
				containerAnalysis,
				config,
				analysisOpts.TimeRange,
			)
			if err != nil {
				return fmt.Errorf("failed to generate container recommendation for %s: %w", container.Name, err)
			}

			containerRecs[i] = rec
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	recs := make([]ContainerRecommendation, 0, len(containers))
	for _, rec := range containerRecs {
		if rec != nil {
			recs = append(recs, *rec)
		}
	}

	return recs, nil
}

func mergeContainerRecommendation(existing, incoming *ContainerRecommendation) {
	cpuIncomingWon := updateMaxQuantity(&existing.CPU.RecommendedRequest, incoming.CPU.RecommendedRequest) ||
		updateMaxQuantity(&existing.CPU.RecommendedLimit, incoming.CPU.RecommendedLimit)
	if cpuIncomingWon {
		existing.CPU.Confidence = incoming.CPU.Confidence
	}

	memoryIncomingWon := updateMaxQuantity(&existing.Memory.RecommendedRequest, incoming.Memory.RecommendedRequest) ||
		updateMaxQuantity(&existing.Memory.RecommendedLimit, incoming.Memory.RecommendedLimit)
	if memoryIncomingWon {
		existing.Memory.Confidence = incoming.Memory.Confidence
	}

	hasCPURecommendation := existing.CPU.RecommendedRequest != nil || existing.CPU.RecommendedLimit != nil
	hasMemoryRecommendation := existing.Memory.RecommendedRequest != nil || existing.Memory.RecommendedLimit != nil
	existing.Confidence = calculateOverallConfidence(
		existing.CPU.Confidence,
		existing.Memory.Confidence,
		hasCPURecommendation,
		hasMemoryRecommendation,
	)

	existing.CPU.RequestChangePercent = calculateChangePercentFromStrings(existing.CPU.CurrentRequest, existing.CPU.RecommendedRequest)
	existing.CPU.LimitChangePercent = calculateChangePercentFromStrings(existing.CPU.CurrentLimit, existing.CPU.RecommendedLimit)
	existing.Memory.RequestChangePercent = calculateChangePercentFromStrings(existing.Memory.CurrentRequest, existing.Memory.RecommendedRequest)
	existing.Memory.LimitChangePercent = calculateChangePercentFromStrings(existing.Memory.CurrentLimit, existing.Memory.RecommendedLimit)
}

// NewComputeRecommendationRecord returns a compute recommendation record with a new UUIDv7.
func NewComputeRecommendationRecord(rec Recommendation) (*database.ComputeRecommendation, error) {
	recommendationsJSON, err := json.Marshal(rec.Recommendations)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal recommendations: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate recommendation id: %w", err)
	}

	now := time.Now()
	return &database.ComputeRecommendation{
		ID:                 id,
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

func calculateOverallConfidence(
	cpuConfidence float64,
	memoryConfidence float64,
	hasCPURec bool,
	hasMemoryRec bool,
) float64 {
	if hasCPURec && hasMemoryRec {
		return max(0.0, min(1.0, (cpuConfidence+memoryConfidence)/2))
	}
	if hasCPURec {
		return max(0.0, min(1.0, cpuConfidence))
	}
	if hasMemoryRec {
		return max(0.0, min(1.0, memoryConfidence))
	}
	return 0
}

// finalizeResourceRecommendations enforces Guaranteed QoS (request equals limit), fills a missing limit
// from request when needed, and avoids recommending a limit below an existing request.
func finalizeResourceRecommendations(
	recommendedRequest *float64,
	recommendedLimit *float64,
	currentRequest *float64,
	currentLimit *float64,
	mode RecommendationMode,
) (*float64, *float64) {
	if mode == ModeGuaranteed {
		var guaranteedValue float64
		if recommendedRequest != nil {
			guaranteedValue = *recommendedRequest
		} else if currentRequest != nil && *currentRequest > 0 {
			guaranteedValue = *currentRequest
		} else if recommendedLimit != nil {
			guaranteedValue = *recommendedLimit
		} else if currentLimit != nil && *currentLimit > 0 {
			guaranteedValue = *currentLimit
		} else {
			return nil, nil
		}

		return &guaranteedValue, &guaranteedValue
	}

	if recommendedLimit == nil && recommendedRequest != nil {
		if currentLimit != nil && *currentLimit >= *recommendedRequest {
			return recommendedRequest, currentLimit
		}
		return recommendedRequest, recommendedRequest
	}

	if recommendedRequest == nil && recommendedLimit != nil {
		if currentRequest != nil && *recommendedLimit < *currentRequest {
			return currentRequest, currentRequest
		}
		if currentRequest != nil {
			return currentRequest, recommendedLimit
		}
		return recommendedLimit, recommendedLimit
	}

	return recommendedRequest, recommendedLimit
}

// formatCPUQuantity renders cores as a Kubernetes CPU string (millicores below 1 core, else three decimals).
func formatCPUQuantity(cores float64) string {
	cores = max(cores, 0)

	if cores < 1.0 {
		millicores := max(int64(cores*1000), 0)
		return fmt.Sprintf("%dm", millicores)
	}
	return fmt.Sprintf("%.3f", cores)
}

// formatMemoryQuantity picks Ki/Mi/Gi/Ti units rounded to a whole unit for Kubernetes quantity strings.
func formatMemoryQuantity(bytes int64) string {
	bytes = max(bytes, 0)

	const (
		KiB = 1024
		MiB = 1024 * KiB
		GiB = 1024 * MiB
		TiB = 1024 * GiB
	)

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

func updateMaxQuantity(target **string, source *string) bool {
	if source == nil {
		return false
	}
	if *target == nil {
		*target = source
		return true
	}

	q1, err1 := resource.ParseQuantity(**target)
	q2, err2 := resource.ParseQuantity(*source)
	if err1 != nil || err2 != nil {
		return false
	}

	if q2.Cmp(q1) > 0 {
		*target = source
		return true
	}
	return false
}

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

// calculateChangePercent returns nil if recommended is nil, 100% when current is missing or zero,
// otherwise the relative percent change rounded to one decimal.
func calculateChangePercent(current, recommended *float64) *float64 {
	if recommended == nil {
		return nil
	}

	if current == nil || *current == 0 {
		return new(100.0)
	}

	changePercent := ((*recommended - *current) / *current) * 100.0

	return new(math.Round(changePercent*10) / 10)
}
