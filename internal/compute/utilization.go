package compute

import (
	"fmt"

	"github.com/thread_koder/mochi/internal/timeseries"
)

// Represents utilization time series
type TimeSeries struct {
	CPU    []timeseries.DataPoint `json:"cpu"`
	Memory []timeseries.DataPoint `json:"memory"`
}

// Represents resource metrics (raw data)
type ResourceMetrics struct {
	CPU            []timeseries.DataPoint `json:"cpu"`
	Memory         []timeseries.DataPoint `json:"memory"`
	CPUThrottling  []timeseries.DataPoint `json:"cpu_throttling,omitempty"`
	CPUPressure    []timeseries.DataPoint `json:"cpu_pressure,omitempty"`
	MemoryFailCnt  []timeseries.DataPoint `json:"memory_fail_cnt,omitempty"`
	MemoryOOM      []timeseries.DataPoint `json:"memory_oom,omitempty"`
	MemoryPressure []timeseries.DataPoint `json:"memory_pressure,omitempty"`
	Restarts       []timeseries.DataPoint `json:"restarts,omitempty"`
}

// Represents CPU utilization analysis results
type CPUUtilization struct {
	Current   float64                  `json:"current"`   // Most recent value
	Stats     timeseries.StatsResult   `json:"stats"`     // Contains Mean (average), Max (peak), Min, etc.
	Trend     timeseries.TrendResult   `json:"trend"`     // Trend analysis (increasing/decreasing/stable)
	Anomalies timeseries.AnomalyResult `json:"anomalies"` // Detected anomalies (outliers)
}

// Represents memory utilization analysis results
type MemoryUtilization struct {
	Current   float64                  `json:"current"`   // Most recent value (bytes)
	Stats     timeseries.StatsResult   `json:"stats"`     // Contains Mean (average), Max (peak), Min, etc.
	Trend     timeseries.TrendResult   `json:"trend"`     // Trend analysis (increasing/decreasing/stable)
	Anomalies timeseries.AnomalyResult `json:"anomalies"` // Detected anomalies (outliers)
}

// Represents overall utilization analysis results
type UtilizationResult struct {
	CPU    CPUUtilization    `json:"cpu"`
	Memory MemoryUtilization `json:"memory"`
}

// Analyzes CPU utilization from time series data
func AnalyzeCPUUtilization(cpuData []timeseries.DataPoint) (CPUUtilization, error) {
	if len(cpuData) == 0 {
		return CPUUtilization{}, fmt.Errorf("cannot analyze CPU utilization from empty dataset")
	}

	// Calculate current (most recent value)
	current := cpuData[len(cpuData)-1].Value

	// Calculate stats
	stats, err := timeseries.CalculateStats(cpuData)
	if err != nil {
		return CPUUtilization{}, fmt.Errorf("failed to calculate CPU stats: %w", err)
	}

	// Analyze trend
	trend, err := timeseries.AnalyzeTrend(cpuData)
	if err != nil {
		// If trend analysis fails (e.g., insufficient data), use stable trend
		trend = timeseries.TrendResult{
			Direction: timeseries.DirectionStable,
			Slope:     0,
			Strength:  0,
		}
	}

	// Detect anomalies
	anomalies, err := timeseries.DetectAnomalies(cpuData, 4.0) // 4 standard deviations
	if err != nil {
		// If anomaly detection fails, use empty result
		anomalies = timeseries.AnomalyResult{
			Anomalies:    []timeseries.Anomaly{},
			AnomalyCount: 0,
			Threshold:    0,
		}
	}

	return CPUUtilization{
		Current:   current,
		Stats:     stats,
		Trend:     trend,
		Anomalies: anomalies,
	}, nil
}

// Analyzes memory utilization from time series data
func AnalyzeMemoryUtilization(memoryData []timeseries.DataPoint) (MemoryUtilization, error) {
	if len(memoryData) == 0 {
		return MemoryUtilization{}, fmt.Errorf("cannot analyze memory utilization from empty dataset")
	}

	// Calculate current (most recent value)
	current := memoryData[len(memoryData)-1].Value

	// Calculate stats
	stats, err := timeseries.CalculateStats(memoryData)
	if err != nil {
		return MemoryUtilization{}, fmt.Errorf("failed to calculate memory stats: %w", err)
	}

	// Analyze trend
	trend, err := timeseries.AnalyzeTrend(memoryData)
	if err != nil {
		// If trend analysis fails (e.g., insufficient data), use stable trend
		trend = timeseries.TrendResult{
			Direction: timeseries.DirectionStable,
			Slope:     0,
			Strength:  0,
		}
	}

	// Detect anomalies
	anomalies, err := timeseries.DetectAnomalies(memoryData, 4.0) // 4 standard deviations
	if err != nil {
		// If anomaly detection fails, use empty result
		anomalies = timeseries.AnomalyResult{
			Anomalies:    []timeseries.Anomaly{},
			AnomalyCount: 0,
			Threshold:    0,
		}
	}

	return MemoryUtilization{
		Current:   current,
		Stats:     stats,
		Trend:     trend,
		Anomalies: anomalies,
	}, nil
}

// Analyzes resource utilization from metrics
func AnalyzeUtilization(metrics ResourceMetrics) (UtilizationResult, error) {
	// Validate we have metrics
	if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
		return UtilizationResult{}, fmt.Errorf("no metrics provided for utilization analysis")
	}

	var result UtilizationResult
	var err error

	// Analyze CPU
	if len(metrics.CPU) > 0 {
		result.CPU, err = AnalyzeCPUUtilization(metrics.CPU)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze CPU utilization: %w", err)
		}
	}

	// Analyze memory
	if len(metrics.Memory) > 0 {
		result.Memory, err = AnalyzeMemoryUtilization(metrics.Memory)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze memory utilization: %w", err)
		}
	}

	return result, nil
}
