package disk

import (
	"fmt"

	"github.com/thread_koder/mochi/internal/timeseries"
)

// TimeSeries holds read/write byte-rate and operation-rate samples for charting.
type TimeSeries struct {
	ReadBytes  []timeseries.DataPoint `json:"read_bytes"`
	WriteBytes []timeseries.DataPoint `json:"write_bytes"`
	ReadOps    []timeseries.DataPoint `json:"read_ops"`
	WriteOps   []timeseries.DataPoint `json:"write_ops"`
}

// DirectionUtilization summarizes one disk direction (read or write) for bytes
// or operations.
type DirectionUtilization struct {
	Current   float64                  `json:"current"` // latest sample (bytes/sec or ops/sec)
	Stats     timeseries.StatsResult   `json:"stats"`
	Trend     timeseries.TrendResult   `json:"trend"`
	Anomalies timeseries.AnomalyResult `json:"anomalies"`
}

// UtilizationResult joins read/write byte and operation summaries.
type UtilizationResult struct {
	ReadBytes  DirectionUtilization `json:"read_bytes"`
	WriteBytes DirectionUtilization `json:"write_bytes"`
	ReadOps    DirectionUtilization `json:"read_ops"`
	WriteOps   DirectionUtilization `json:"write_ops"`
}

const anomalyStdDevs = 4.0

// AnalyzeReadBytesUtilization summarizes read byte-rate samples. Trend or
// anomaly steps return stable or empty results when the series is too thin, so
// stats and Current still surface.
func AnalyzeReadBytesUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze read bytes utilization from empty dataset")
	}

	current := data[len(data)-1].Value

	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return DirectionUtilization{}, fmt.Errorf("failed to calculate read bytes stats: %w", err)
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

// AnalyzeWriteBytesUtilization summarizes write byte-rate samples. Thin series
// degrade trend and anomaly fields the same way as AnalyzeReadBytesUtilization.
func AnalyzeWriteBytesUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze write bytes utilization from empty dataset")
	}

	current := data[len(data)-1].Value

	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return DirectionUtilization{}, fmt.Errorf("failed to calculate write bytes stats: %w", err)
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

// AnalyzeReadOpsUtilization summarizes read operation-rate samples. Thin series
// degrade trend and anomaly fields the same way as AnalyzeReadBytesUtilization.
func AnalyzeReadOpsUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze read ops utilization from empty dataset")
	}

	current := data[len(data)-1].Value

	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return DirectionUtilization{}, fmt.Errorf("failed to calculate read ops stats: %w", err)
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

// AnalyzeWriteOpsUtilization summarizes write operation-rate samples. Thin
// series degrade trend and anomaly fields the same way as AnalyzeReadOpsUtilization.
func AnalyzeWriteOpsUtilization(data []timeseries.DataPoint) (DirectionUtilization, error) {
	if len(data) == 0 {
		return DirectionUtilization{}, fmt.Errorf("cannot analyze write ops utilization from empty dataset")
	}

	current := data[len(data)-1].Value

	stats, err := timeseries.CalculateStats(data)
	if err != nil {
		return DirectionUtilization{}, fmt.Errorf("failed to calculate write ops stats: %w", err)
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

// AnalyzeUtilization fills UtilizationResult from DiskMetrics. At least one of
// read or write byte series must be non-empty, and operation series may be empty.
func AnalyzeUtilization(metrics DiskMetrics) (UtilizationResult, error) {
	if len(metrics.ReadBytes) == 0 && len(metrics.WriteBytes) == 0 {
		return UtilizationResult{}, fmt.Errorf("no metrics provided for utilization analysis")
	}

	var result UtilizationResult
	var err error

	if len(metrics.ReadBytes) > 0 {
		result.ReadBytes, err = AnalyzeReadBytesUtilization(metrics.ReadBytes)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze read bytes utilization: %w", err)
		}
	}

	if len(metrics.WriteBytes) > 0 {
		result.WriteBytes, err = AnalyzeWriteBytesUtilization(metrics.WriteBytes)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze write bytes utilization: %w", err)
		}
	}

	if len(metrics.ReadOps) > 0 {
		result.ReadOps, err = AnalyzeReadOpsUtilization(metrics.ReadOps)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze read ops utilization: %w", err)
		}
	}

	if len(metrics.WriteOps) > 0 {
		result.WriteOps, err = AnalyzeWriteOpsUtilization(metrics.WriteOps)
		if err != nil {
			return UtilizationResult{}, fmt.Errorf("failed to analyze write ops utilization: %w", err)
		}
	}

	return result, nil
}
