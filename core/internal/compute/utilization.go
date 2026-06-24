package compute

import (
	"fmt"

	"github.com/thread_koder/mochi/core/internal/timeseries"
)

// TimeSeries is raw CPU and memory series used for charting.
type TimeSeries struct {
	CPU    []timeseries.DataPoint `json:"cpu"`
	Memory []timeseries.DataPoint `json:"memory"`
}

// CPUUtilization summarizes CPU usage from a time series: latest value, distribution stats, trend, and anomalies.
type CPUUtilization struct {
	Current    float64                  `json:"current"`
	Stats      timeseries.StatsResult   `json:"stats"`
	Trend      timeseries.TrendResult   `json:"trend"`
	Anomalies  timeseries.AnomalyResult `json:"anomalies"`
	SampleSize int                      `json:"sample_size"`
}

// MemoryUtilization summarizes memory usage (bytes) from a time series: latest value, stats, trend, and anomalies.
type MemoryUtilization struct {
	Current    float64                  `json:"current"`
	Stats      timeseries.StatsResult   `json:"stats"`
	Trend      timeseries.TrendResult   `json:"trend"`
	Anomalies  timeseries.AnomalyResult `json:"anomalies"`
	SampleSize int                      `json:"sample_size"`
}

// UtilizationResult is CPU and memory utilization summaries for one scope (container, pod, workload, or namespace).
type UtilizationResult struct {
	CPU    CPUUtilization    `json:"cpu"`
	Memory MemoryUtilization `json:"memory"`
}

const anomalyStdDevs = 4.0

// AnalyzeCPUUtilization derives stats, trend, and anomalies from CPU series. Trend and anomaly steps
// return neutral empty results when the series is too short or noisy so callers can still use stats.
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
		trend = timeseries.TrendResult{
			Direction: timeseries.DirectionStable,
			Slope:     0,
			Strength:  0,
		}
	}

	anomalies, err := timeseries.DetectAnomalies(cpuData, anomalyStdDevs)
	if err != nil {
		anomalies = timeseries.AnomalyResult{
			Anomalies:    []timeseries.Anomaly{},
			AnomalyCount: 0,
			Threshold:    0,
		}
	}

	return CPUUtilization{
		Current:    current,
		Stats:      stats,
		Trend:      trend,
		Anomalies:  anomalies,
		SampleSize: len(cpuData),
	}, nil
}

// AnalyzeMemoryUtilization derives stats, trend, and anomalies from memory series. Trend and anomaly steps
// return neutral empty results when the series is too short or noisy so callers can still use stats.
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
		trend = timeseries.TrendResult{
			Direction: timeseries.DirectionStable,
			Slope:     0,
			Strength:  0,
		}
	}

	anomalies, err := timeseries.DetectAnomalies(memoryData, anomalyStdDevs)
	if err != nil {
		anomalies = timeseries.AnomalyResult{
			Anomalies:    []timeseries.Anomaly{},
			AnomalyCount: 0,
			Threshold:    0,
		}
	}

	return MemoryUtilization{
		Current:    current,
		Stats:      stats,
		Trend:      trend,
		Anomalies:  anomalies,
		SampleSize: len(memoryData),
	}, nil
}

// AnalyzeUtilization runs CPU and/or memory analysis when the corresponding series are non-empty.
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
