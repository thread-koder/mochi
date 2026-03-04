package network

import (
	"fmt"

	"github.com/thread_koder/mochi/internal/timeseries"
)

// Represents network I/O time series for charting
type TimeSeries struct {
	ReceiveBytes  []timeseries.DataPoint `json:"receive_bytes"`
	TransmitBytes []timeseries.DataPoint `json:"transmit_bytes"`
}

// Represents raw network metrics data
type NetworkMetrics struct {
	ReceiveBytes    []timeseries.DataPoint `json:"receive_bytes"`
	TransmitBytes   []timeseries.DataPoint `json:"transmit_bytes"`
	ReceiveErrors   []timeseries.DataPoint `json:"receive_errors"`
	TransmitErrors  []timeseries.DataPoint `json:"transmit_errors"`
	ReceiveDropped  []timeseries.DataPoint `json:"receive_dropped"`
	TransmitDropped []timeseries.DataPoint `json:"transmit_dropped"`
}

// Represents analysis results for a single direction (receive or transmit)
type DirectionUtilization struct {
	Current   float64                  `json:"current"`   // Most recent value (bytes/sec)
	Stats     timeseries.StatsResult   `json:"stats"`     // Contains Mean, Max, Min, Percentiles, etc.
	Trend     timeseries.TrendResult   `json:"trend"`     // Trend analysis (increasing/decreasing/stable)
	Anomalies timeseries.AnomalyResult `json:"anomalies"` // Detected anomalies (outliers)
}

// Represents analysis results for errors in a single direction
type ErrorsResult struct {
	Current float64 `json:"current"` // Most recent value (errors/sec or packets/sec)
	Total   float64 `json:"total"`   // Sum over the analysis period
}

// Represents overall network utilization analysis results
type UtilizationResult struct {
	Receive         DirectionUtilization `json:"receive"`
	Transmit        DirectionUtilization `json:"transmit"`
	ReceiveErrors   ErrorsResult         `json:"receive_errors"`
	TransmitErrors  ErrorsResult         `json:"transmit_errors"`
	ReceiveDropped  ErrorsResult         `json:"receive_dropped"`
	TransmitDropped ErrorsResult         `json:"transmit_dropped"`
}

// Analyzes receive bytes utilization from time series data
func AnalyzeReceiveUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze receive utilization from empty dataset")
	}

	// Calculate current (most recent value)
	current := data[len(data)-1].Value

	// Calculate stats
	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return DirectionUtilization{}, fmt.Errorf("failed to calculate receive stats: %w", err)
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

// Analyzes transmit bytes utilization from time series data
func AnalyzeTransmitUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze transmit utilization from empty dataset")
	}

	// Calculate current (most recent value)
	current := data[len(data)-1].Value

	// Calculate stats
	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return DirectionUtilization{}, fmt.Errorf("failed to calculate transmit stats: %w", err)
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

// Analyzes errors/dropped packets from time series data
func AnalyzeErrors(data []timeseries.DataPoint) ErrorsResult {
	if len(data) == 0 {
		return ErrorsResult{Current: 0, Total: 0}
	}

	current := data[len(data)-1].Value
	var total float64
	for _, dp := range data {
		total += dp.Value
	}

	return ErrorsResult{
		Current: current,
		Total:   total,
	}
}

// Analyzes network utilization from metrics
func AnalyzeUtilization(metrics NetworkMetrics) (UtilizationResult, error) {
	// Validate we have at least receive or transmit metrics
	if len(metrics.ReceiveBytes) == 0 && len(metrics.TransmitBytes) == 0 {
		return UtilizationResult{}, fmt.Errorf("no metrics provided for utilization analysis")
	}

	var result UtilizationResult
	var err error

	// Analyze receive bytes
	if len(metrics.ReceiveBytes) > 0 {
		result.Receive, err = AnalyzeReceiveUtilization(metrics.ReceiveBytes)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze receive utilization: %w", err)
		}
	}

	// Analyze transmit bytes
	if len(metrics.TransmitBytes) > 0 {
		result.Transmit, err = AnalyzeTransmitUtilization(metrics.TransmitBytes)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze transmit utilization: %w", err)
		}
	}

	// Analyze errors and dropped packets
	result.ReceiveErrors = AnalyzeErrors(metrics.ReceiveErrors)
	result.TransmitErrors = AnalyzeErrors(metrics.TransmitErrors)
	result.ReceiveDropped = AnalyzeErrors(metrics.ReceiveDropped)
	result.TransmitDropped = AnalyzeErrors(metrics.TransmitDropped)

	return result, nil
}
