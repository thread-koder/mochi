package timeseries

import "time"

// DataPoint is one timestamped sample in a metric time series.
type DataPoint struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}
