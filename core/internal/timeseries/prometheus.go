package timeseries

import (
	"sort"

	"github.com/prometheus/common/model"
)

// MatrixToDataPoints flattens a Prometheus matrix into a single time-ordered slice.
// Callers aggregate by timestamp later when multiple series represent one workload metric.
func MatrixToDataPoints(matrix model.Matrix) []DataPoint {
	totalSize := 0
	for _, series := range matrix {
		totalSize += len(series.Values)
	}

	dataPoints := make([]DataPoint, 0, totalSize)

	for _, series := range matrix {
		for _, sample := range series.Values {
			dataPoints = append(dataPoints, DataPoint{
				Value:     float64(sample.Value),
				Timestamp: sample.Timestamp.Time(),
			})
		}
	}

	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Timestamp.Before(dataPoints[j].Timestamp)
	})

	return dataPoints
}

// VectorToDataPoints converts an instant vector into DataPoint values.
func VectorToDataPoints(vector model.Vector) []DataPoint {
	dataPoints := make([]DataPoint, 0, len(vector))
	for _, sample := range vector {
		dataPoints = append(dataPoints, DataPoint{
			Value:     float64(sample.Value),
			Timestamp: sample.Timestamp.Time(),
		})
	}
	return dataPoints
}
