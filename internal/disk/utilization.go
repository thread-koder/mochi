package disk

import (
	"fmt"

	"github.com/thread_koder/mochi/internal/timeseries"
)

// Represents disk I/O time series for charting
type TimeSeries struct {
	ReadBytes  []timeseries.DataPoint `json:"read_bytes"`
	WriteBytes []timeseries.DataPoint `json:"write_bytes"`
	ReadOps    []timeseries.DataPoint `json:"read_ops"`
	WriteOps   []timeseries.DataPoint `json:"write_ops"`
}

// Represents analysis results for a single direction (read or write)
type DirectionUtilization struct {
	Current   float64                  `json:"current"`   // Most recent value (bytes/sec or ops/sec)
	Stats     timeseries.StatsResult   `json:"stats"`     // Contains Mean, Max, Min, Percentiles, etc.
	Trend     timeseries.TrendResult   `json:"trend"`     // Trend analysis (increasing/decreasing/stable)
	Anomalies timeseries.AnomalyResult `json:"anomalies"` // Detected anomalies (outliers)
}

// Represents overall disk utilization analysis results
type UtilizationResult struct {
	ReadBytes  DirectionUtilization `json:"read_bytes"`
	WriteBytes DirectionUtilization `json:"write_bytes"`
	ReadOps    DirectionUtilization `json:"read_ops"`
	WriteOps   DirectionUtilization `json:"write_ops"`
}

// Analyzes read bytes utilization from time series data
func AnalyzeReadBytesUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze read bytes utilization from empty dataset")
	}

	// Calculate current (most recent value)
	current := data[len(data)-1].Value

	// Calculate stats
	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return DirectionUtilization{}, fmt.Errorf("failed to calculate read bytes stats: %w", err)
	}

	// Analyze trend
	trend, err := timeseries.AnalyzeTrend(data)
	if err != nil {
		// If trend analysis fails (e.g., insufficient data), use stable trend
		trend = timeseries.TrendResult{
			Direction: timeseries.DirectionStable,
			Slope:     0,
			Strength:  0,
		}
	}

	// Detect anomalies
	anomalies, err := timeseries.DetectAnomalies(data, 4.0) // 4 standard deviations
	if err != nil {
		// If anomaly detection fails, use empty result
		anomalies = timeseries.AnomalyResult{
			Anomalies:    []timeseries.Anomaly{},
			AnomalyCount: 0,
			Threshold:    0,
		}
	}

	return DirectionUtilization{
		Current:   current,
		Stats:     stats,
		Trend:     trend,
		Anomalies: anomalies,
	}, nil
}

// Analyzes write bytes utilization from time series data
func AnalyzeWriteBytesUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze write bytes utilization from empty dataset")
	}

	// Calculate current (most recent value)
	current := data[len(data)-1].Value

	// Calculate stats
	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return DirectionUtilization{}, fmt.Errorf("failed to calculate write bytes stats: %w", err)
	}

	// Analyze trend
	trend, err := timeseries.AnalyzeTrend(data)
	if err != nil {
		// If trend analysis fails (e.g., insufficient data), use stable trend
		trend = timeseries.TrendResult{
			Direction: timeseries.DirectionStable,
			Slope:     0,
			Strength:  0,
		}
	}

	// Detect anomalies
	anomalies, err := timeseries.DetectAnomalies(data, 4.0) // 4 standard deviations
	if err != nil {
		// If anomaly detection fails, use empty result
		anomalies = timeseries.AnomalyResult{
			Anomalies:    []timeseries.Anomaly{},
			AnomalyCount: 0,
			Threshold:    0,
		}
	}

	return DirectionUtilization{
		Current:   current,
		Stats:     stats,
		Trend:     trend,
		Anomalies: anomalies,
	}, nil
}

// Analyzes read operations utilization from time series data
func AnalyzeReadOpsUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze read ops utilization from empty dataset")
	}

	// Calculate current (most recent value)
	current := data[len(data)-1].Value

	// Calculate stats
	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return DirectionUtilization{}, fmt.Errorf("failed to calculate read ops stats: %w", err)
	}

	// Analyze trend
	trend, err := timeseries.AnalyzeTrend(data)
	if err != nil {
		// If trend analysis fails (e.g., insufficient data), use stable trend
		trend = timeseries.TrendResult{
			Direction: timeseries.DirectionStable,
			Slope:     0,
			Strength:  0,
		}
	}

	// Detect anomalies
	anomalies, err := timeseries.DetectAnomalies(data, 4.0) // 4 standard deviations
	if err != nil {
		// If anomaly detection fails, use empty result
		anomalies = timeseries.AnomalyResult{
			Anomalies:    []timeseries.Anomaly{},
			AnomalyCount: 0,
			Threshold:    0,
		}
	}

	return DirectionUtilization{
		Current:   current,
		Stats:     stats,
		Trend:     trend,
		Anomalies: anomalies,
	}, nil
}

// Analyzes write operations utilization from time series data
func AnalyzeWriteOpsUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze write ops utilization from empty dataset")
	}

	// Calculate current (most recent value)
	current := data[len(data)-1].Value

	// Calculate stats
	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return DirectionUtilization{}, fmt.Errorf("failed to calculate write ops stats: %w", err)
	}

	// Analyze trend
	trend, err := timeseries.AnalyzeTrend(data)
	if err != nil {
		// If trend analysis fails (e.g., insufficient data), use stable trend
		trend = timeseries.TrendResult{
			Direction: timeseries.DirectionStable,
			Slope:     0,
			Strength:  0,
		}
	}

	// Detect anomalies
	anomalies, err := timeseries.DetectAnomalies(data, 4.0) // 4 standard deviations
	if err != nil {
		// If anomaly detection fails, use empty result
		anomalies = timeseries.AnomalyResult{
			Anomalies:    []timeseries.Anomaly{},
			AnomalyCount: 0,
			Threshold:    0,
		}
	}

	return DirectionUtilization{
		Current:   current,
		Stats:     stats,
		Trend:     trend,
		Anomalies: anomalies,
	}, nil
}

// Analyzes disk utilization from metrics
func AnalyzeUtilization(metrics DiskMetrics) (UtilizationResult, error) {
	// Validate we have at least read or write bytes metrics
	if len(metrics.ReadBytes) == 0 && len(metrics.WriteBytes) == 0 {
		return UtilizationResult{}, fmt.Errorf("no metrics provided for utilization analysis")
	}

	var result UtilizationResult
	var err error

	// Analyze read bytes
	if len(metrics.ReadBytes) > 0 {
		result.ReadBytes, err = AnalyzeReadBytesUtilization(metrics.ReadBytes)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze read bytes utilization: %w", err)
		}
	}

	// Analyze write bytes
	if len(metrics.WriteBytes) > 0 {
		result.WriteBytes, err = AnalyzeWriteBytesUtilization(metrics.WriteBytes)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze write bytes utilization: %w", err)
		}
	}

	// Analyze read ops
	if len(metrics.ReadOps) > 0 {
		result.ReadOps, err = AnalyzeReadOpsUtilization(metrics.ReadOps)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze read ops utilization: %w", err)
		}
	}

	// Analyze write ops
	if len(metrics.WriteOps) > 0 {
		result.WriteOps, err = AnalyzeWriteOpsUtilization(metrics.WriteOps)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze write ops utilization: %w", err)
		}
	}

	return result, nil
}
