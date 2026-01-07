package analyzer

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/prometheus/common/model"
)

// Represents the trend direction
type Direction string

const (
	DirectionStable     Direction = "stable"
	DirectionIncreasing Direction = "increasing"
	DirectionDecreasing Direction = "decreasing"
)

// Represents the severity level of an anomaly
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

// Represents a data point in a time series
type DataPoint struct {
	Value     float64
	Timestamp time.Time
}

// Represents percentile calculation results
type PercentileResult struct {
	P50 float64
	P95 float64
	P99 float64
}

// Represents trend analysis results
type TrendResult struct {
	Direction Direction // "increasing", "decreasing", or "stable"
	Slope     float64   // Linear regression slope
	Strength  float64   // Correlation coefficient (0-1)
}

// Represents peak detection results
type PeakResult struct {
	Peaks     []Peak
	PeakCount int
	MaxPeak   float64
	MinPeak   float64
	AvgPeak   float64
}

// Represents a detected peak
type Peak struct {
	Value     float64
	Timestamp time.Time
	Index     int
}

// Represents statistical calculation results
type StatsResult struct {
	Mean       float64
	Median     float64
	StdDev     float64
	Min        float64
	Max        float64
	Percentile PercentileResult
}

// Represents anomaly detection results
type AnomalyResult struct {
	Anomalies    []Anomaly
	AnomalyCount int
	Threshold    float64
}

// Represents a detected anomaly
type Anomaly struct {
	Value     float64
	Timestamp time.Time
	Index     int
	Deviation float64  // How many standard deviations from mean
	Severity  Severity // "low", "medium", "high"
}

// Calculates percentiles (P50, P95, P99) from a slice of values
func CalculatePercentiles(values []float64) (PercentileResult, error) {
	if len(values) == 0 {
		return PercentileResult{}, fmt.Errorf("cannot calculate percentiles from empty dataset")
	}

	// Create a copy to avoid modifying the original slice
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

// Calculates a specific percentile from a sorted slice
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	if len(sorted) == 1 {
		return sorted[0]
	}

	// Linear interpolation method
	index := (p / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// Analyzes trend direction and strength from time series data
func AnalyzeTrend(dataPoints []DataPoint) (TrendResult, error) {
	if len(dataPoints) < 2 {
		return TrendResult{}, fmt.Errorf("need at least 2 data points for trend analysis")
	}

	// Extract values and timestamps
	values := make([]float64, len(dataPoints))
	timestamps := make([]float64, len(dataPoints))

	for i, dp := range dataPoints {
		values[i] = dp.Value
		timestamps[i] = float64(dp.Timestamp.Unix())
	}

	// Calculate linear regression
	slope, _, correlation := linearRegression(timestamps, values)

	// Determine direction
	var direction Direction
	if math.Abs(slope) < 1e-10 {
		direction = DirectionStable
	} else if slope > 0 {
		direction = DirectionIncreasing
	} else {
		direction = DirectionDecreasing
	}

	// Calculate strength (absolute correlation coefficient)
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

// Performs linear regression on two datasets
func linearRegression(x, y []float64) (slope, intercept, correlation float64) {
	n := len(x)
	if n != len(y) || n == 0 {
		return 0, 0, 0
	}

	// Calculate means
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

	// Calculate slope and intercept
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

	// Calculate correlation coefficient
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

// Detects peaks in time series data using a simple local maximum approach
func DetectPeaks(dataPoints []DataPoint, windowSize int) (PeakResult, error) {
	if len(dataPoints) == 0 {
		return PeakResult{}, fmt.Errorf("cannot detect peaks from empty dataset")
	}

	if windowSize < 1 {
		windowSize = 3 // Default window size
	}

	if windowSize > len(dataPoints)/2 {
		newSize := max(len(dataPoints)/2, 1)
		windowSize = newSize
	}

	peaks := make([]Peak, 0)
	values := make([]float64, len(dataPoints))

	for i, dp := range dataPoints {
		values[i] = dp.Value
	}

	// Handle very small datasets
	minRequiredSize := 2*windowSize + 1
	if len(values) < minRequiredSize {
		return PeakResult{
			Peaks:     []Peak{},
			PeakCount: 0,
			MaxPeak:   0,
			MinPeak:   0,
			AvgPeak:   0,
		}, nil
	}

	// Detect local maxima
	for i := windowSize; i < len(values)-windowSize; i++ {
		isPeak := true
		currentValue := values[i]

		// Check if current point is higher than all points in the window
		for j := i - windowSize; j <= i+windowSize; j++ {
			if j != i && values[j] >= currentValue {
				isPeak = false
				break
			}
		}

		if isPeak {
			peaks = append(peaks, Peak{
				Value:     currentValue,
				Timestamp: dataPoints[i].Timestamp,
				Index:     i,
			})
		}
	}

	// Calculate statistics
	var maxPeak, minPeak, sumPeak float64
	if len(peaks) > 0 {
		maxPeak = peaks[0].Value
		minPeak = peaks[0].Value
		for _, peak := range peaks {
			if peak.Value > maxPeak {
				maxPeak = peak.Value
			}
			if peak.Value < minPeak {
				minPeak = peak.Value
			}
			sumPeak += peak.Value
		}
	}

	avgPeak := 0.0
	if len(peaks) > 0 {
		avgPeak = sumPeak / float64(len(peaks))
	}

	return PeakResult{
		Peaks:     peaks,
		PeakCount: len(peaks),
		MaxPeak:   maxPeak,
		MinPeak:   minPeak,
		AvgPeak:   avgPeak,
	}, nil
}

// Calculates statistical summary from time series data
func CalculateStats(dataPoints []DataPoint) (StatsResult, error) {
	if len(dataPoints) == 0 {
		return StatsResult{}, fmt.Errorf("cannot calculate stats from empty dataset")
	}

	values := make([]float64, len(dataPoints))
	min := dataPoints[0].Value
	max := dataPoints[0].Value

	// Extract values and calculate min/max
	for i, dp := range dataPoints {
		values[i] = dp.Value
		if dp.Value < min {
			min = dp.Value
		}
		if dp.Value > max {
			max = dp.Value
		}
	}

	// Calculate basic statistics
	mean := calculateMean(values)
	median := calculateMedian(values)
	stdDev := calculateStdDev(values, mean)

	// Calculate percentiles
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

// Detects anomalies in time series data using statistical methods
func DetectAnomalies(dataPoints []DataPoint, thresholdMultiplier float64) (AnomalyResult, error) {
	if len(dataPoints) == 0 {
		return AnomalyResult{}, fmt.Errorf("cannot detect anomalies from empty dataset")
	}

	if thresholdMultiplier <= 0 {
		thresholdMultiplier = 3.0 // Default: 3 standard deviations
	}

	// Calculate stats
	stats, err := CalculateStats(dataPoints)
	if err != nil {
		return AnomalyResult{}, fmt.Errorf("failed to calculate stats: %w", err)
	}

	// Skip anomaly detection if standard deviation is zero (all values are the same)
	if stats.StdDev == 0 {
		return AnomalyResult{
			Anomalies:    []Anomaly{},
			AnomalyCount: 0,
			Threshold:    0,
		}, nil
	}

	threshold := thresholdMultiplier * stats.StdDev
	anomalies := make([]Anomaly, 0)

	// Detect anomalies
	for i, dp := range dataPoints {
		deviation := math.Abs(dp.Value - stats.Mean)
		stdDevs := deviation / stats.StdDev

		// Check if value exceeds threshold
		if deviation > threshold {
			severity := SeverityLow
			if stdDevs > 3.0 {
				severity = SeverityHigh
			} else if stdDevs > 2.0 {
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

// Converts Prometheus Matrix to DataPoint slice
func MatrixToDataPoints(matrix model.Matrix) []DataPoint {
	// Calculate total capacity to avoid reallocations
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

	return dataPoints
}

// Converts Prometheus Vector to DataPoint slice
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

// Calculates the mean of a slice of values
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// Calculates the median of a slice of values
func calculateMedian(values []float64) float64 {
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

// Calculates the standard deviation of a slice of values
func calculateStdDev(values []float64, mean float64) float64 {
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
