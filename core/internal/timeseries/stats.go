package timeseries

import (
	"fmt"
	"math"
	"sort"
)

// PercentileResult contains the percentile values.
type PercentileResult struct {
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

// StatsResult is the statistical summary derived from a data series.
type StatsResult struct {
	Mean       float64          `json:"mean"`
	Median     float64          `json:"median"`
	StdDev     float64          `json:"std_dev"`
	Min        float64          `json:"min"`
	Max        float64          `json:"max"`
	Percentile PercentileResult `json:"percentile"`
}

// CalculatePercentiles computes P50, P95, and P99 from the provided values.
func CalculatePercentiles(values []float64) (PercentileResult, error) {
	if len(values) == 0 {
		return PercentileResult{}, fmt.Errorf("cannot calculate percentiles from empty dataset")
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	p50 := percentile(sorted, 50)
	p95 := percentile(sorted, 95)
	p99 := percentile(sorted, 99)

	return PercentileResult{
		P50: p50,
		P95: p95,
		P99: p99,
	}, nil
}

func CalculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// ExtractValues copies only the numeric values from data points.
func ExtractValues(dataPoints []DataPoint) []float64 {
	values := make([]float64, len(dataPoints))
	for i, dp := range dataPoints {
		values[i] = dp.Value
	}
	return values
}

// CalculateStats calculates a full summary (central tendency, spread, and percentiles).
func CalculateStats(dataPoints []DataPoint) (StatsResult, error) {
	if len(dataPoints) == 0 {
		return StatsResult{}, fmt.Errorf("cannot calculate stats from empty dataset")
	}

	values := make([]float64, len(dataPoints))
	min := dataPoints[0].Value
	max := dataPoints[0].Value

	for i, dp := range dataPoints {
		values[i] = dp.Value
		if dp.Value < min {
			min = dp.Value
		}
		if dp.Value > max {
			max = dp.Value
		}
	}

	mean := CalculateMean(values)
	median := median(values)
	stdDev := stdDev(values, mean)

	percentiles, err := CalculatePercentiles(values)
	if err != nil {
		return StatsResult{}, fmt.Errorf("failed to calculate percentiles: %w", err)
	}

	return StatsResult{
		Mean:       mean,
		Median:     median,
		StdDev:     stdDev,
		Min:        min,
		Max:        max,
		Percentile: percentiles,
	}, nil
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	index := (p / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2.0
	}
	return sorted[n/2]
}

func stdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sumSqDiff float64
	for _, v := range values {
		diff := v - mean
		sumSqDiff += diff * diff
	}
	variance := sumSqDiff / float64(len(values))
	return math.Sqrt(variance)
}
