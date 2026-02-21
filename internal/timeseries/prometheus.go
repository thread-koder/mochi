package timeseries

import (
	"sort"

	"github.com/prometheus/common/model"
)

// Converts a Prometheus Matrix to a DataPoint slice
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

// Converts a Prometheus Vector to a DataPoint slice
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
