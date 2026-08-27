package network

import (
	"fmt"

	"github.com/thread_koder/mochi/core/internal/timeseries"
)

type TimeSeries struct {
	ReceiveBytes  []timeseries.DataPoint `json:"receive_bytes"`
	TransmitBytes []timeseries.DataPoint `json:"transmit_bytes"`
}

type DirectionUtilization struct {
	Current    float64                  `json:"current"`
	Stats      timeseries.StatsResult   `json:"stats"`
	Trend      timeseries.TrendResult   `json:"trend"`
	Anomalies  timeseries.AnomalyResult `json:"anomalies"`
	SampleSize int                      `json:"sample_size"`
}

type ErrorsResult struct {
	Current float64 `json:"current"`
	Total   float64 `json:"total"`
}

type UtilizationResult struct {
	Receive         DirectionUtilization `json:"receive"`
	Transmit        DirectionUtilization `json:"transmit"`
	ReceiveErrors   ErrorsResult         `json:"receive_errors"`
	TransmitErrors  ErrorsResult         `json:"transmit_errors"`
	ReceiveDropped  ErrorsResult         `json:"receive_dropped"`
	TransmitDropped ErrorsResult         `json:"transmit_dropped"`
}

const anomalyStdDevs = 4.0

func hasAnalyzableNetworkMetrics(metrics NetworkMetrics) bool {
	return timeseries.HasEnoughPoints(metrics.ReceiveBytes) || timeseries.HasEnoughPoints(metrics.TransmitBytes)
}

func analyzeDirectionUtilization(data []timeseries.DataPoint, label string) (DirectionUtilization, error) {
	if !timeseries.HasEnoughPoints(data) {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze %s utilization from dataset with fewer than %d points", label, timeseries.MinPointsForAnalysis)
	}

	current := data[len(data)-1].Value

	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return DirectionUtilization{}, err
	}

	trend, err := timeseries.AnalyzeTrend(data)
	if err != nil {
		return DirectionUtilization{}, err
	}

	anomalies, err := timeseries.DetectAnomalies(data, anomalyStdDevs)
	if err != nil {
		return DirectionUtilization{}, err
	}

	return DirectionUtilization{
		Current:    current,
		Stats:      stats,
		Trend:      trend,
		Anomalies:  anomalies,
		SampleSize: len(data),
	}, nil
}

func analyzeErrors(data []timeseries.DataPoint) ErrorsResult {
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

func AnalyzeUtilization(metrics NetworkMetrics) (UtilizationResult, error) {
	if !hasAnalyzableNetworkMetrics(metrics) {
		return UtilizationResult{}, fmt.Errorf("no metrics provided for utilization analysis")
	}

	var result UtilizationResult
	var err error

	if timeseries.HasEnoughPoints(metrics.ReceiveBytes) {
		result.Receive, err = analyzeDirectionUtilization(metrics.ReceiveBytes, "receive")
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	if timeseries.HasEnoughPoints(metrics.TransmitBytes) {
		result.Transmit, err = analyzeDirectionUtilization(metrics.TransmitBytes, "transmit")
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	result.ReceiveErrors = analyzeErrors(metrics.ReceiveErrors)
	result.TransmitErrors = analyzeErrors(metrics.TransmitErrors)
	result.ReceiveDropped = analyzeErrors(metrics.ReceiveDropped)
	result.TransmitDropped = analyzeErrors(metrics.TransmitDropped)

	return result, nil
}
