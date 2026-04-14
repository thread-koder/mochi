package timeseries

import (
	"sort"
	"time"
)

// DataPoint is one timestamped sample in a metric time series.
type DataPoint struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// MergeDataPointsByTime combines two series by exact timestamp and sums values that land
// on the same instant. Callers use this to collapse per-resource series into a single view.
func MergeDataPointsByTime(existing []DataPoint, incoming []DataPoint) []DataPoint {
	if len(existing) == 0 {
		return AggregateDataPointsByTimestamp(incoming)
	}

	valuesByTime := make(map[time.Time]float64)
	for _, dp := range existing {
		valuesByTime[dp.Timestamp] += dp.Value
	}
	for _, dp := range incoming {
		valuesByTime[dp.Timestamp] += dp.Value
	}

	return TimeMapToSortedDataPoints(valuesByTime)
}

// AggregateDataPointsByTimestamp collapses duplicate timestamps within one slice by summing
// their values. This keeps downstream stats/trend code working on one point per timestamp.
func AggregateDataPointsByTimestamp(dataPoints []DataPoint) []DataPoint {
	if len(dataPoints) == 0 {
		return dataPoints
	}

	valuesByTime := make(map[time.Time]float64)
	for _, dp := range dataPoints {
		valuesByTime[dp.Timestamp] += dp.Value
	}

	return TimeMapToSortedDataPoints(valuesByTime)
}

// TimeMapToSortedDataPoints converts a timestamp map into a chronologically sorted slice.
func TimeMapToSortedDataPoints(valuesByTime map[time.Time]float64) []DataPoint {
	result := make([]DataPoint, 0, len(valuesByTime))
	for ts, val := range valuesByTime {
		result = append(result, DataPoint{
			Timestamp: ts,
			Value:     val,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	return result
}
