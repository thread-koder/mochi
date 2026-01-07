package analyzer

import (
	"fmt"
)

// Represents resource utilization metrics
type ResourceMetrics struct {
	CPU    []DataPoint
	Memory []DataPoint
}

// Represents CPU utilization analysis results
type CPUUtilization struct {
	Current   float64       // Most recent value
	Stats     StatsResult   // Contains Mean (average), Max (peak), Min, etc.
	Trend     TrendResult   // Trend analysis (increasing/decreasing/stable)
	Anomalies AnomalyResult // Detected anomalies (outliers)
}

// Represents memory utilization analysis results
type MemoryUtilization struct {
	Current   float64       // Most recent value (bytes)
	Stats     StatsResult   // Contains Mean (average), Max (peak), Min, etc.
	Trend     TrendResult   // Trend analysis (increasing/decreasing/stable)
	Anomalies AnomalyResult // Detected anomalies (outliers)
}

// Represents overall utilization analysis results
type UtilizationResult struct {
	CPU    CPUUtilization
	Memory MemoryUtilization
}

// Analyzes CPU utilization from time series data
func AnalyzeCPUUtilization(cpuData []DataPoint) (CPUUtilization, error) {
	if len(cpuData) == 0 {
		return CPUUtilization{}, fmt.Errorf("cannot analyze CPU utilization from empty dataset")
	}

	// Calculate current (most recent value)
	current := cpuData[len(cpuData)-1].Value

	// Calculate stats
	stats, err := CalculateStats(cpuData)
	if err != nil {
		return CPUUtilization{}, fmt.Errorf("failed to calculate CPU stats: %w", err)
	}

	// Analyze trend
	trend, err := AnalyzeTrend(cpuData)
	if err != nil {
		// If trend analysis fails (e.g., insufficient data), use stable trend
		trend = TrendResult{
			Direction: DirectionStable,
			Slope:     0,
			Strength:  0,
		}
	}

	// Detect anomalies
	anomalies, err := DetectAnomalies(cpuData, 3.0) // 3 standard deviations
	if err != nil {
		// If anomaly detection fails, use empty result
		anomalies = AnomalyResult{
			Anomalies:    []Anomaly{},
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
func AnalyzeMemoryUtilization(memoryData []DataPoint) (MemoryUtilization, error) {
	if len(memoryData) == 0 {
		return MemoryUtilization{}, fmt.Errorf("cannot analyze memory utilization from empty dataset")
	}

	// Calculate current (most recent value)
	current := memoryData[len(memoryData)-1].Value

	// Calculate stats
	stats, err := CalculateStats(memoryData)
	if err != nil {
		return MemoryUtilization{}, fmt.Errorf("failed to calculate memory stats: %w", err)
	}

	// Analyze trend
	trend, err := AnalyzeTrend(memoryData)
	if err != nil {
		// If trend analysis fails (e.g., insufficient data), use stable trend
		trend = TrendResult{
			Direction: DirectionStable,
			Slope:     0,
			Strength:  0,
		}
	}

	// Detect anomalies
	anomalies, err := DetectAnomalies(memoryData, 3.0) // 3 standard deviations
	if err != nil {
		// If anomaly detection fails, use empty result
		anomalies = AnomalyResult{
			Anomalies:    []Anomaly{},
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
