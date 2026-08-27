package compute

import (
	"fmt"

	"github.com/thread_koder/mochi/core/internal/timeseries"
)

type TimeSeries struct {
	CPU    []timeseries.DataPoint `json:"cpu"`
	Memory []timeseries.DataPoint `json:"memory"`
}

type ResourceUtilization struct {
	Current    float64                  `json:"current"`
	Stats      timeseries.StatsResult   `json:"stats"`
	Trend      timeseries.TrendResult   `json:"trend"`
	Anomalies  timeseries.AnomalyResult `json:"anomalies"`
	SampleSize int                      `json:"sample_size"`
}

type UtilizationResult struct {
	CPU    ResourceUtilization `json:"cpu"`
	Memory ResourceUtilization `json:"memory"`
}

const anomalyStdDevs = 4.0

func hasAnalyzableComputeMetrics(metrics ResourceMetrics) bool {
	return timeseries.HasEnoughPoints(metrics.CPU) || timeseries.HasEnoughPoints(metrics.Memory)
}

func analyzeResourceUtilization(data []timeseries.DataPoint, label string) (ResourceUtilization, error) {
	if !timeseries.HasEnoughPoints(data) {
		return ResourceUtilization{}, fmt.Errorf("cannot analyze %s utilization from dataset with fewer than %d points", label, timeseries.MinPointsForAnalysis)
	}

	current := data[len(data)-1].Value

	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return ResourceUtilization{}, err
	}

	trend, err := timeseries.AnalyzeTrend(data)
	if err != nil {
		return ResourceUtilization{}, err
	}

	anomalies, err := timeseries.DetectAnomalies(data, anomalyStdDevs)
	if err != nil {
		return ResourceUtilization{}, err
	}

	return ResourceUtilization{
		Current:    current,
		Stats:      stats,
		Trend:      trend,
		Anomalies:  anomalies,
		SampleSize: len(data),
	}, nil
}

func AnalyzeUtilization(metrics ResourceMetrics) (UtilizationResult, error) {
	if !hasAnalyzableComputeMetrics(metrics) {
		return UtilizationResult{}, fmt.Errorf("no metrics provided for utilization analysis")
	}

	var result UtilizationResult
	var err error

	if timeseries.HasEnoughPoints(metrics.CPU) {
		result.CPU, err = analyzeResourceUtilization(metrics.CPU, "CPU")
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	if timeseries.HasEnoughPoints(metrics.Memory) {
		result.Memory, err = analyzeResourceUtilization(metrics.Memory, "memory")
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	return result, nil
}
