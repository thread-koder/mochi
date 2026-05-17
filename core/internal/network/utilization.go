package network

import (
	"fmt"

	"github.com/thread_koder/mochi/core/internal/timeseries"
)

// TimeSeries holds receive and transmit byte-rate samples for charting.
type TimeSeries struct {
	ReceiveBytes  []timeseries.DataPoint `json:"receive_bytes"`
	TransmitBytes []timeseries.DataPoint `json:"transmit_bytes"`
}

// DirectionUtilization summarizes one byte-rate direction (receive or transmit).
type DirectionUtilization struct {
	Current   float64                  `json:"current"` // latest sample (bytes/sec)
	Stats     timeseries.StatsResult   `json:"stats"`
	Trend     timeseries.TrendResult   `json:"trend"`
	Anomalies timeseries.AnomalyResult `json:"anomalies"`
}

// ErrorsResult summarizes error or drop counters for one direction at the latest step and across the window.
type ErrorsResult struct {
	Current float64 `json:"current"`
	Total   float64 `json:"total"`
}

// UtilizationResult joins byte-rate directions with error and drop counter summaries.
type UtilizationResult struct {
	Receive         DirectionUtilization `json:"receive"`
	Transmit        DirectionUtilization `json:"transmit"`
	ReceiveErrors   ErrorsResult         `json:"receive_errors"`
	TransmitErrors  ErrorsResult         `json:"transmit_errors"`
	ReceiveDropped  ErrorsResult         `json:"receive_dropped"`
	TransmitDropped ErrorsResult         `json:"transmit_dropped"`
}

const anomalyStdDevs = 4.0

// AnalyzeReceiveUtilization summarizes receive byte-rate samples. Trend or anomaly steps return
// stable or empty results when the series is too thin, so stats and Current still surface.
func AnalyzeReceiveUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze receive utilization from empty dataset")
	}

	current := data[len(data)-1].Value

	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return DirectionUtilization{}, fmt.Errorf("failed to calculate receive stats: %w", err)
	}

	trend, err := timeseries.AnalyzeTrend(data)
	if err != nil {
		trend = timeseries.TrendResult{
			Direction: timeseries.DirectionStable,
			Slope:     0,
			Strength:  0,
		}
	}

	anomalies, err := timeseries.DetectAnomalies(data, anomalyStdDevs)
	if err != nil {
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

// AnalyzeTransmitUtilization summarizes transmit byte-rate samples. Thin series degrade trend and
// anomaly fields the same way as AnalyzeReceiveUtilization.
func AnalyzeTransmitUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze transmit utilization from empty dataset")
	}

	current := data[len(data)-1].Value

	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return DirectionUtilization{}, fmt.Errorf("failed to calculate transmit stats: %w", err)
	}

	trend, err := timeseries.AnalyzeTrend(data)
	if err != nil {
		trend = timeseries.TrendResult{
			Direction: timeseries.DirectionStable,
			Slope:     0,
			Strength:  0,
		}
	}

	anomalies, err := timeseries.DetectAnomalies(data, anomalyStdDevs)
	if err != nil {
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

// AnalyzeErrors returns the latest sample as Current and the sum of step values as Total.
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

// AnalyzeUtilization fills UtilizationResult from NetworkMetrics. At least one of receive or
// transmit byte series must be non-empty, and error and drop series may be empty and yield zeros.
func AnalyzeUtilization(metrics NetworkMetrics) (UtilizationResult, error) {
	if len(metrics.ReceiveBytes) == 0 && len(metrics.TransmitBytes) == 0 {
		return UtilizationResult{}, fmt.Errorf("no metrics provided for utilization analysis")
	}

	var result UtilizationResult
	var err error

	if len(metrics.ReceiveBytes) > 0 {
		result.Receive, err = AnalyzeReceiveUtilization(metrics.ReceiveBytes)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze receive utilization: %w", err)
		}
	}

	if len(metrics.TransmitBytes) > 0 {
		result.Transmit, err = AnalyzeTransmitUtilization(metrics.TransmitBytes)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze transmit utilization: %w", err)
		}
	}

	result.ReceiveErrors = AnalyzeErrors(metrics.ReceiveErrors)
	result.TransmitErrors = AnalyzeErrors(metrics.TransmitErrors)
	result.ReceiveDropped = AnalyzeErrors(metrics.ReceiveDropped)
	result.TransmitDropped = AnalyzeErrors(metrics.TransmitDropped)

	return result, nil
}
