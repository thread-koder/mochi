package timeseries

import (
	"fmt"
	"math"
)

type Direction string

const (
	DirectionStable     Direction = "stable"
	DirectionIncreasing Direction = "increasing"
	DirectionDecreasing Direction = "decreasing"
)

type TrendResult struct {
	Direction Direction `json:"direction"`
	Slope     float64   `json:"slope"`
	Strength  float64   `json:"strength"`
}

// AnalyzeTrend estimates direction and strength with simple linear regression.
func AnalyzeTrend(dataPoints []DataPoint) (TrendResult, error) {
	if len(dataPoints) < 2 {
		return TrendResult{}, fmt.Errorf("need at least 2 data points for trend analysis")
	}

	values := make([]float64, len(dataPoints))
	timestamps := make([]float64, len(dataPoints))

	for i, dp := range dataPoints {
		values[i] = dp.Value
		timestamps[i] = float64(dp.Timestamp.Unix())
	}

	slope, _, correlation := linearRegression(timestamps, values)

	var direction Direction
	if math.Abs(slope) < 1e-10 {
		direction = DirectionStable
	} else if slope > 0 {
		direction = DirectionIncreasing
	} else {
		direction = DirectionDecreasing
	}

	strength := math.Abs(correlation)
	if strength > 1.0 {
		strength = 1.0
	}

	return TrendResult{
		Direction: direction,
		Slope:     slope,
		Strength:  strength,
	}, nil
}

func linearRegression(x, y []float64) (slope, intercept, correlation float64) {
	n := len(x)
	if n != len(y) || n == 0 {
		return 0, 0, 0
	}

	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for i := range n {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	var numerator, denominator float64
	for i := range n {
		numerator += (x[i] - meanX) * (y[i] - meanY)
		denominator += (x[i] - meanX) * (x[i] - meanX)
	}

	if denominator == 0 {
		return 0, meanY, 0
	}

	slope = numerator / denominator
	intercept = meanY - slope*meanX

	var sumSqX, sumSqY float64
	for i := range n {
		dx := x[i] - meanX
		dy := y[i] - meanY
		sumSqX += dx * dx
		sumSqY += dy * dy
	}

	if sumSqX == 0 || sumSqY == 0 {
		correlation = 0
	} else {
		correlation = numerator / math.Sqrt(sumSqX*sumSqY)
	}

	return slope, intercept, correlation
}
