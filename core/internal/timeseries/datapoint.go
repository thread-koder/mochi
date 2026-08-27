package timeseries

import "time"

type DataPoint struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// MinPointsForAnalysis is the minimum samples required to publish utilization
// (stats + trend + anomalies). Below this, a series is treated as absent.
const MinPointsForAnalysis = 2

func HasEnoughPoints(dataPoints []DataPoint) bool {
	return len(dataPoints) >= MinPointsForAnalysis
}
