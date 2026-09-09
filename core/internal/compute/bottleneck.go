package compute

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/thread_koder/mochi/core/internal/analyzer"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
	"golang.org/x/sync/errgroup"
)

type BottleneckResource string

const (
	BottleneckResourceCPU    BottleneckResource = "cpu"
	BottleneckResourceMemory BottleneckResource = "memory"
)

type BottleneckKind string

const (
	BottleneckCPUThrottled         BottleneckKind = "cpu_throttled"
	BottleneckCPUPressure          BottleneckKind = "cpu_pressure"
	BottleneckCPULimitSaturated    BottleneckKind = "cpu_limit_saturated"
	BottleneckMemoryOOM            BottleneckKind = "memory_oom"
	BottleneckMemoryFailCnt        BottleneckKind = "memory_failcnt"
	BottleneckMemoryPressure       BottleneckKind = "memory_pressure"
	BottleneckMemoryLimitSaturated BottleneckKind = "memory_limit_saturated"
)

// kindSeverity ranks primary kinds so replica merge keeps the worse signal.
var kindSeverity = map[BottleneckKind]int{
	BottleneckCPUThrottled:         3,
	BottleneckCPUPressure:          2,
	BottleneckCPULimitSaturated:    1,
	BottleneckMemoryOOM:            4,
	BottleneckMemoryFailCnt:        3,
	BottleneckMemoryPressure:       2,
	BottleneckMemoryLimitSaturated: 1,
}

type BottleneckFinding struct {
	WorkloadType           string             `json:"workload_type"`
	WorkloadName           string             `json:"workload_name"`
	ContainerName          string             `json:"container_name"`
	Resource               BottleneckResource `json:"resource"`
	Kind                   BottleneckKind     `json:"kind"`
	Score                  float64            `json:"score"`
	CPUThrottling          float64            `json:"cpu_throttling"`
	CPUPressure            float64            `json:"cpu_pressure"`
	MemoryOOM              float64            `json:"memory_oom"`
	MemoryFailCnt          float64            `json:"memory_fail_cnt"`
	MemoryPressure         float64            `json:"memory_pressure"`
	CPULimitUtilization    float64            `json:"cpu_limit_utilization"`
	MemoryLimitUtilization float64            `json:"memory_limit_utilization"`
}

type NamespaceBottlenecks struct {
	Namespace string              `json:"namespace"`
	TimeRange string              `json:"time_range"`
	Findings  []BottleneckFinding `json:"findings"`
}

type workloadBottleneckResult struct {
	Findings   []BottleneckFinding
	HadMetrics bool
}

// ClassifyContainerBottlenecks picks at most one CPU and one memory runtime bottleneck
// from existing container analysis.
func ClassifyContainerBottlenecks(analysis ContainerAnalysis) []BottleneckFinding {
	stability := analysis.Stability
	cpuProv := analysis.Provisioning.CPU
	memProv := analysis.Provisioning.Memory

	base := BottleneckFinding{
		ContainerName:  analysis.ContainerName,
		CPUThrottling:  stability.CPUThrottling,
		CPUPressure:    stability.CPUPressure,
		MemoryOOM:      stability.MemoryOOM,
		MemoryFailCnt:  stability.MemoryFailCnt,
		MemoryPressure: stability.MemoryPressure,
	}
	if cpuProv.LimitUtilization != nil {
		base.CPULimitUtilization = *cpuProv.LimitUtilization
	}
	if memProv.LimitUtilization != nil {
		base.MemoryLimitUtilization = *memProv.LimitUtilization
	}

	findings := make([]BottleneckFinding, 0, 2)

	if kind, score, ok := classifyCPUBottleneck(stability, cpuProv); ok {
		finding := base
		finding.Resource = BottleneckResourceCPU
		finding.Kind = kind
		finding.Score = score
		findings = append(findings, finding)
	}

	if kind, score, ok := classifyMemoryBottleneck(stability, memProv); ok {
		finding := base
		finding.Resource = BottleneckResourceMemory
		finding.Kind = kind
		finding.Score = score
		findings = append(findings, finding)
	}

	return findings
}

func classifyCPUBottleneck(stability StabilityResult, prov ResourceProvisioning) (BottleneckKind, float64, bool) {
	if stability.CPUThrottling > ThrottlingThreshold {
		penalty := min((stability.CPUThrottling-ThrottlingThreshold)*2.0, maxPenaltyCPUThrottling)
		return BottleneckCPUThrottled, penalty / maxPenaltyCPUThrottling, true
	}
	if stability.CPUPressure > PressureThreshold {
		penalty := min((stability.CPUPressure-PressureThreshold)*0.5, maxPenaltyCPUPressure)
		return BottleneckCPUPressure, penalty / maxPenaltyCPUPressure, true
	}
	if prov.CurrentLimit != nil && *prov.CurrentLimit > 0 && *prov.LimitUtilization > (1.0-CPUHeadroom) {
		return BottleneckCPULimitSaturated, limitSaturationScore(*prov.LimitUtilization, CPUHeadroom), true
	}
	return "", 0, false
}

func classifyMemoryBottleneck(stability StabilityResult, prov ResourceProvisioning) (BottleneckKind, float64, bool) {
	if stability.MemoryOOM > 0 {
		penalty := eventCountPenalty(stability.MemoryOOM, maxPenaltyOOM, oomCountAtMax)
		return BottleneckMemoryOOM, penalty / maxPenaltyOOM, true
	}
	if stability.MemoryFailCnt > 0 {
		penalty := eventCountPenalty(stability.MemoryFailCnt, maxPenaltyMemoryFailCnt, memoryFailCountAtMax)
		return BottleneckMemoryFailCnt, penalty / maxPenaltyMemoryFailCnt, true
	}
	if stability.MemoryPressure > PressureThreshold {
		penalty := min((stability.MemoryPressure-PressureThreshold)*1.0, maxPenaltyMemoryPressure)
		return BottleneckMemoryPressure, penalty / maxPenaltyMemoryPressure, true
	}
	if prov.CurrentLimit != nil && *prov.CurrentLimit > 0 && *prov.LimitUtilization > (1.0-MemoryHeadroom) {
		return BottleneckMemoryLimitSaturated, limitSaturationScore(*prov.LimitUtilization, MemoryHeadroom), true
	}
	return "", 0, false
}

// limitSaturationScore mirrors provisioning limitPenalty inverted onto 0–1 severity.
func limitSaturationScore(limitUtilization, headroom float64) float64 {
	return max(0.0, min(1.0, 1.0-((1.0-limitUtilization)/headroom)))
}

func GetNamespaceBottlenecks(ctx context.Context, namespace string, opts AnalysisOptions) (NamespaceBottlenecks, error) {
	if err := opts.Validate(); err != nil {
		return NamespaceBottlenecks{}, fmt.Errorf("invalid analysis options: %w", err)
	}

	opts.IncludeTimeSeries = false
	since := time.Now().Add(-opts.TimeRange)

	results, err := analyzer.AnalyzeWorkloads(ctx, namespace, since,
		func(ctx context.Context, kind, name, _ string, pods database.PodsForAnalysis) (workloadBottleneckResult, error) {
			return bottlenecksForWorkload(ctx, kind, name, pods, opts)
		})
	if err != nil {
		return NamespaceBottlenecks{}, fmt.Errorf("failed to get bottlenecks for namespace %s: %w", namespace, err)
	}

	hadMetrics := false
	findings := make([]BottleneckFinding, 0)
	for _, result := range results {
		if result.HadMetrics {
			hadMetrics = true
		}
		findings = append(findings, result.Findings...)
	}

	if len(results) > 0 && !hadMetrics {
		return NamespaceBottlenecks{}, apperrors.NewNoMetrics(fmt.Sprintf("namespace %s", namespace))
	}

	sortBottleneckFindings(findings)

	return NamespaceBottlenecks{
		Namespace: namespace,
		TimeRange: opts.TimeRange.String(),
		Findings:  findings,
	}, nil
}

func bottlenecksForWorkload(
	ctx context.Context,
	workloadType string,
	workloadName string,
	pods database.PodsForAnalysis,
	opts AnalysisOptions,
) (workloadBottleneckResult, error) {
	podFindings := make([][]BottleneckFinding, len(pods.All))
	podHadMetrics := make([]bool, len(pods.All))
	g, gctx := errgroup.WithContext(ctx)
	for i, pod := range pods.All {
		g.Go(func() error {
			findings, hadMetrics, err := bottlenecksForPod(gctx, pod, opts)
			if err != nil {
				return err
			}
			podFindings[i] = findings
			podHadMetrics[i] = hadMetrics
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return workloadBottleneckResult{}, err
	}

	hadMetrics := false
	findings := make([]BottleneckFinding, 0)
	for i, pf := range podFindings {
		if podHadMetrics[i] {
			hadMetrics = true
		}
		for _, finding := range pf {
			finding.WorkloadType = workloadType
			finding.WorkloadName = workloadName
			findings = append(findings, finding)
		}
	}

	return workloadBottleneckResult{
		Findings:   mergeBottleneckFindings(findings),
		HadMetrics: hadMetrics,
	}, nil
}

func bottlenecksForPod(
	ctx context.Context,
	pod *database.Pod,
	opts AnalysisOptions,
) ([]BottleneckFinding, bool, error) {
	containers, err := database.GetContainersForAnalysis(ctx, pod.UID)
	if err != nil {
		return nil, false, err
	}

	analyzed, err := analyzer.SkipNoMetrics(ctx, containers, func(ctx context.Context, container *database.Container) ([]BottleneckFinding, error) {
		analysis, err := AnalyzeContainer(ctx, container, opts)
		if err != nil {
			return nil, err
		}
		return ClassifyContainerBottlenecks(analysis), nil
	})
	if err != nil {
		return nil, false, err
	}

	findings := make([]BottleneckFinding, 0)
	for _, containerFindings := range analyzed {
		findings = append(findings, containerFindings...)
	}

	return findings, len(analyzed) > 0, nil
}

func mergeBottleneckFindings(findings []BottleneckFinding) []BottleneckFinding {
	type mergeKey struct {
		containerName string
		resource      BottleneckResource
	}

	merged := make(map[mergeKey]BottleneckFinding, len(findings))
	for _, finding := range findings {
		key := mergeKey{
			containerName: finding.ContainerName,
			resource:      finding.Resource,
		}
		existing, ok := merged[key]
		if !ok || bottleneckWorse(finding, existing) {
			merged[key] = finding
		}
	}

	results := make([]BottleneckFinding, 0, len(merged))
	for _, finding := range merged {
		results = append(results, finding)
	}
	return results
}

func bottleneckWorse(a, b BottleneckFinding) bool {
	sevA := kindSeverity[a.Kind]
	sevB := kindSeverity[b.Kind]
	if sevA != sevB {
		return sevA > sevB
	}
	return a.Score > b.Score
}

func sortBottleneckFindings(findings []BottleneckFinding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Score != findings[j].Score {
			return findings[i].Score > findings[j].Score
		}
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		if findings[i].WorkloadType != findings[j].WorkloadType {
			return findings[i].WorkloadType < findings[j].WorkloadType
		}
		if findings[i].WorkloadName != findings[j].WorkloadName {
			return findings[i].WorkloadName < findings[j].WorkloadName
		}
		return findings[i].ContainerName < findings[j].ContainerName
	})
}
