package timeseries

import (
	"fmt"
	"math"
	"time"
)

// Severity labels how far an outlier is from the baseline distribution.
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

// Anomaly represents one outlier detected in a series.
type Anomaly struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Index     int       `json:"index"`
	Deviation float64   `json:"deviation"`
	Severity  Severity  `json:"severity"`
}

// AnomalyResult contains all detected anomalies and the threshold used.
type AnomalyResult struct {
	Anomalies    []Anomaly `json:"anomalies"`
	AnomalyCount int       `json:"anomaly_count"`
	Threshold    float64   `json:"threshold"`
}

// DetectAnomalies flags points whose deviation clears both an absolute threshold
// (N * stddev) and a relative-change guardrail to avoid noisy tiny baselines.
func DetectAnomalies(dataPoints []DataPoint, thresholdMultiplier float64) (AnomalyResult, error) {
	if len(dataPoints) == 0 {
		return AnomalyResult{}, fmt.Errorf("cannot detect anomalies from empty dataset")
	}

	if thresholdMultiplier <= 0 {
		thresholdMultiplier = 3.0
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

		// The relative guardrail avoids marking tiny absolute wiggles as anomalies.
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
