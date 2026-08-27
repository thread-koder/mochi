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
	Current    float64                  `json:"current"`
	Stats      timeseries.StatsResult   `json:"stats"`
	Trend      timeseries.TrendResult   `json:"trend"`
	Anomalies  timeseries.AnomalyResult `json:"anomalies"`
	SampleSize int                      `json:"sample_size"`
}

type UtilizationResult struct {
	ReadBytes  DirectionUtilization `json:"read_bytes"`
	WriteBytes DirectionUtilization `json:"write_bytes"`
	ReadOps    DirectionUtilization `json:"read_ops"`
	WriteOps   DirectionUtilization `json:"write_ops"`
}

const anomalyStdDevs = 4.0

func hasAnalyzableDiskMetrics(metrics DiskMetrics) bool {
	return timeseries.HasEnoughPoints(metrics.ReadBytes) ||
		timeseries.HasEnoughPoints(metrics.WriteBytes) ||
		timeseries.HasEnoughPoints(metrics.ReadOps) ||
		timeseries.HasEnoughPoints(metrics.WriteOps)
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

func AnalyzeUtilization(metrics DiskMetrics) (UtilizationResult, error) {
	if !hasAnalyzableDiskMetrics(metrics) {
		return UtilizationResult{}, fmt.Errorf("no metrics provided for utilization analysis")
	}

	var result UtilizationResult
	var err error

	if timeseries.HasEnoughPoints(metrics.ReadBytes) {
		result.ReadBytes, err = analyzeDirectionUtilization(metrics.ReadBytes, "read bytes")
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	if timeseries.HasEnoughPoints(metrics.WriteBytes) {
		result.WriteBytes, err = analyzeDirectionUtilization(metrics.WriteBytes, "write bytes")
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	if timeseries.HasEnoughPoints(metrics.ReadOps) {
		result.ReadOps, err = analyzeDirectionUtilization(metrics.ReadOps, "read ops")
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	if timeseries.HasEnoughPoints(metrics.WriteOps) {
		result.WriteOps, err = analyzeDirectionUtilization(metrics.WriteOps, "write ops")
		if err != nil {
			return UtilizationResult{}, err
		}
	}

	return result, nil
}
