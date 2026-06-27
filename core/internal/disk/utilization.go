package disk

import (
	"fmt"

	"github.com/thread_koder/mochi/core/internal/timeseries"
)

type TimeSeries struct {
	ReadBytes  []timeseries.DataPoint `json:"read_bytes"`
	WriteBytes []timeseries.DataPoint `json:"write_bytes"`
	ReadOps    []timeseries.DataPoint `json:"read_ops"`
	WriteOps   []timeseries.DataPoint `json:"write_ops"`
}

type DirectionUtilization struct {
	Current   float64                  `json:"current"`
	Stats     timeseries.StatsResult   `json:"stats"`
	Trend     timeseries.TrendResult   `json:"trend"`
	Anomalies timeseries.AnomalyResult `json:"anomalies"`
}

type UtilizationResult struct {
	ReadBytes  DirectionUtilization `json:"read_bytes"`
	WriteBytes DirectionUtilization `json:"write_bytes"`
	ReadOps    DirectionUtilization `json:"read_ops"`
	WriteOps   DirectionUtilization `json:"write_ops"`
}

const anomalyStdDevs = 4.0

func AnalyzeReadBytesUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze read bytes utilization from empty dataset")
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

func AnalyzeWriteBytesUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze write bytes utilization from empty dataset")
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

func AnalyzeReadOpsUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze read ops utilization from empty dataset")
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

func AnalyzeWriteOpsUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze write ops utilization from empty dataset")
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

func AnalyzeUtilization(metrics DiskMetrics) (UtilizationResult, error) {
	if len(metrics.ReadBytes) == 0 && len(metrics.WriteBytes) == 0 {
		return UtilizationResult{}, fmt.Errorf("no metrics provided for utilization analysis")
	}

	var result UtilizationResult
	var err error

	if len(metrics.ReadBytes) > 0 {
		result.ReadBytes, err = AnalyzeReadBytesUtilization(metrics.ReadBytes)
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	if len(metrics.WriteBytes) > 0 {
		result.WriteBytes, err = AnalyzeWriteBytesUtilization(metrics.WriteBytes)
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	if len(metrics.ReadOps) > 0 {
		result.ReadOps, err = AnalyzeReadOpsUtilization(metrics.ReadOps)
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	if len(metrics.WriteOps) > 0 {
		result.WriteOps, err = AnalyzeWriteOpsUtilization(metrics.WriteOps)
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	return result, nil
}
