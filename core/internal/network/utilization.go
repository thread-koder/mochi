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
	Current   float64                  `json:"current"`
	Stats     timeseries.StatsResult   `json:"stats"`
	Trend     timeseries.TrendResult   `json:"trend"`
	Anomalies timeseries.AnomalyResult `json:"anomalies"`
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

func AnalyzeReceiveUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze receive utilization from empty dataset")
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
		Current:   current,
		Stats:     stats,
		Trend:     trend,
		Anomalies: anomalies,
	}, nil
}

func AnalyzeTransmitUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze transmit utilization from empty dataset")
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
		Current:   current,
		Stats:     stats,
		Trend:     trend,
		Anomalies: anomalies,
	}, nil
}

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

func AnalyzeUtilization(metrics NetworkMetrics) (UtilizationResult, error) {
	if len(metrics.ReceiveBytes) == 0 && len(metrics.TransmitBytes) == 0 {
		return UtilizationResult{}, fmt.Errorf("no metrics provided for utilization analysis")
	}

	var result UtilizationResult
	var err error

	if len(metrics.ReceiveBytes) > 0 {
		result.Receive, err = AnalyzeReceiveUtilization(metrics.ReceiveBytes)
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	if len(metrics.TransmitBytes) > 0 {
		result.Transmit, err = AnalyzeTransmitUtilization(metrics.TransmitBytes)
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	result.ReceiveErrors = AnalyzeErrors(metrics.ReceiveErrors)
	result.TransmitErrors = AnalyzeErrors(metrics.TransmitErrors)
	result.ReceiveDropped = AnalyzeErrors(metrics.ReceiveDropped)
	result.TransmitDropped = AnalyzeErrors(metrics.TransmitDropped)

	return result, nil
}
