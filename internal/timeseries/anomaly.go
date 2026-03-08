package timeseries

import (
	"fmt"
	"math"
	"time"
)

// Represents the severity level of an anomaly
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

// Represents a detected anomaly
type Anomaly struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Index     int       `json:"index"`
	Deviation float64   `json:"deviation"` // How many standard deviations from mean
	Severity  Severity  `json:"severity"`  // "low", "medium", "high"
}

// Represents anomaly detection results
type AnomalyResult struct {
	Anomalies    []Anomaly `json:"anomalies"`
	AnomalyCount int       `json:"anomaly_count"`
	Threshold    float64   `json:"threshold"`
}

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
