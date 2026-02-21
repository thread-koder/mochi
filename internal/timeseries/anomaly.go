package timeseries

import (
	"fmt"
	"math"
)

// Detects anomalies in time series data using statistical methods
func DetectAnomalies(dataPoints []DataPoint, thresholdMultiplier float64) (AnomalyResult, error) {
	if len(dataPoints) == 0 {
		return AnomalyResult{}, fmt.Errorf("cannot detect anomalies from empty dataset")
	}

	if thresholdMultiplier <= 0 {
		thresholdMultiplier = 3.0 // Default: 3 standard deviations
	}

	stats, err := CalculateStats(dataPoints)
	if err != nil {
		return AnomalyResult{}, fmt.Errorf("failed to calculate stats: %w", err)
	}

	if stats.StdDev == 0 {
		return AnomalyResult{
			Anomalies:    []Anomaly{},
			AnomalyCount: 0,
			Threshold:    0,
		}, nil
	}

	threshold := thresholdMultiplier * stats.StdDev
	anomalies := make([]Anomaly, 0)

	for i, dp := range dataPoints {
		deviation := math.Abs(dp.Value - stats.Mean)
		stdDevs := deviation / stats.StdDev

		isSignificant := stats.Mean == 0 || (deviation/stats.Mean) > 0.1

		if deviation > threshold && isSignificant {
			severity := SeverityLow
			if stdDevs > 5.0 {
				severity = SeverityHigh
			} else if stdDevs > 4.0 {
				severity = SeverityMedium
			}

			anomalies = append(anomalies, Anomaly{
				Value:     dp.Value,
				Timestamp: dp.Timestamp,
				Index:     i,
				Deviation: stdDevs,
				Severity:  severity,
			})
		}
	}

	return AnomalyResult{
		Anomalies:    anomalies,
		AnomalyCount: len(anomalies),
		Threshold:    threshold,
	}, nil
}
