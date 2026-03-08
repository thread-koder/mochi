package timeseries

import (
	"sort"
	"time"
)

// Represents a data point in a time series
type DataPoint struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// Merges two slices of data points by timestamp (sums values at the same timestamp)
// Used when combining data points from multiple sources (e.g., multiple pods in a workload)
func MergeDataPointsByTime(existing []DataPoint, new []DataPoint) []DataPoint {
	if len(existing) == 0 {
		return AggregateDataPointsByTimestamp(new)
	}

	timeMap := make(map[time.Time]float64)
	for _, dp := range existing {
		timeMap[dp.Timestamp] += dp.Value
	}
	for _, dp := range new {
		timeMap[dp.Timestamp] += dp.Value
	}

	return TimeMapToSortedDataPoints(timeMap)
}

// Aggregates a single slice of data points by timestamp (sums values at the same timestamp)
// Used when a single data source contains multiple series (e.g., multiple containers in a pod)
func AggregateDataPointsByTimestamp(dataPoints []DataPoint) []DataPoint {
	if len(dataPoints) == 0 {
		return dataPoints
	}

	timeMap := make(map[time.Time]float64)
	for _, dp := range dataPoints {
		timeMap[dp.Timestamp] += dp.Value
	}

	return TimeMapToSortedDataPoints(timeMap)
}

// Converts a timestamp->value map to a sorted slice of DataPoints
func TimeMapToSortedDataPoints(timeMap map[time.Time]float64) []DataPoint {
	result := make([]DataPoint, 0, len(timeMap))
	for ts, val := range timeMap {
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
