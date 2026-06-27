package compute

import (
	"fmt"

	"github.com/thread_koder/mochi/core/internal/timeseries"
)

type TimeSeries struct {
	CPU    []timeseries.DataPoint `json:"cpu"`
	Memory []timeseries.DataPoint `json:"memory"`
}

type CPUUtilization struct {
	Current    float64                  `json:"current"`
	Stats      timeseries.StatsResult   `json:"stats"`
	Trend      timeseries.TrendResult   `json:"trend"`
	Anomalies  timeseries.AnomalyResult `json:"anomalies"`
	SampleSize int                      `json:"sample_size"`
}

type MemoryUtilization struct {
	Current    float64                  `json:"current"`
	Stats      timeseries.StatsResult   `json:"stats"`
	Trend      timeseries.TrendResult   `json:"trend"`
	Anomalies  timeseries.AnomalyResult `json:"anomalies"`
	SampleSize int                      `json:"sample_size"`
}

type UtilizationResult struct {
	CPU    CPUUtilization    `json:"cpu"`
	Memory MemoryUtilization `json:"memory"`
}

const anomalyStdDevs = 4.0

func AnalyzeCPUUtilization(cpuData []timeseries.DataPoint) (CPUUtilization, error) {
	if len(cpuData) == 0 {
		return CPUUtilization{}, fmt.Errorf("cannot analyze CPU utilization from empty dataset")
	}

	current := cpuData[len(cpuData)-1].Value

	stats, err := timeseries.CalculateStats(cpuData)
	if err != nil {
		return CPUUtilization{}, err
	}

	trend, err := timeseries.AnalyzeTrend(cpuData)
	if err != nil {
		return CPUUtilization{}, err
	}

	anomalies, err := timeseries.DetectAnomalies(cpuData, anomalyStdDevs)
	if err != nil {
		return CPUUtilization{}, err
	}

	return CPUUtilization{
		Current:    current,
		Stats:      stats,
		Trend:      trend,
		Anomalies:  anomalies,
		SampleSize: len(cpuData),
	}, nil
}

func AnalyzeMemoryUtilization(memoryData []timeseries.DataPoint) (MemoryUtilization, error) {
	if len(memoryData) == 0 {
		return MemoryUtilization{}, fmt.Errorf("cannot analyze memory utilization from empty dataset")
	}

	current := memoryData[len(memoryData)-1].Value

	stats, err := timeseries.CalculateStats(memoryData)
	if err != nil {
		return MemoryUtilization{}, err
	}

	trend, err := timeseries.AnalyzeTrend(memoryData)
	if err != nil {
		return MemoryUtilization{}, err
	}

	anomalies, err := timeseries.DetectAnomalies(memoryData, anomalyStdDevs)
	if err != nil {
		return MemoryUtilization{}, err
	}

	return MemoryUtilization{
		Current:    current,
		Stats:      stats,
		Trend:      trend,
		Anomalies:  anomalies,
		SampleSize: len(memoryData),
	}, nil
}

func AnalyzeUtilization(metrics ResourceMetrics) (UtilizationResult, error) {
	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return UtilizationResult{}, fmt.Errorf("no metrics provided for utilization analysis")
	}

	var result UtilizationResult
	var err error

	if len(metrics.CPU) > 0 {
		result.CPU, err = AnalyzeCPUUtilization(metrics.CPU)
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	if len(metrics.Memory) > 0 {
		result.Memory, err = AnalyzeMemoryUtilization(metrics.Memory)
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	return result, nil
}
