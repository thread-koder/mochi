package timeseries

import "time"

type DataPoint struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}
