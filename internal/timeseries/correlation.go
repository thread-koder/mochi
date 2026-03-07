package timeseries

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// Represents the strength of a correlation
type CorrelationStrength string

const (
	CorrelationStrengthWeak     CorrelationStrength = "weak"
	CorrelationStrengthModerate CorrelationStrength = "moderate"
	CorrelationStrengthStrong   CorrelationStrength = "strong"
)

// Represents the direction of a correlation
type CorrelationDirection string

const (
	CorrelationDirectionPositive CorrelationDirection = "positive"
	CorrelationDirectionNegative CorrelationDirection = "negative"
	CorrelationDirectionNone     CorrelationDirection = "none"
)

// Represents the result of a correlation calculation
type CorrelationResult struct {
	Coefficient float64              `json:"coefficient"` // Pearson r (-1 to +1)
	Strength    CorrelationStrength  `json:"strength"`    // weak, moderate, strong
	Direction   CorrelationDirection `json:"direction"`   // positive, negative, none
	SampleSize  int                  `json:"sample_size"` // Number of aligned data points used
}

// Represents correlation at a specific lag
type LagCorrelation struct {
	Lag         time.Duration     `json:"lag"`
	Correlation CorrelationResult `json:"correlation"`
}

// Represents the result of cross-correlation analysis with lag detection
type CrossCorrelationResult struct {
	MaxCorrelation  CorrelationResult `json:"max_correlation"`  // Correlation at optimal lag
	OptimalLag      time.Duration     `json:"optimal_lag"`      // Lag that produces max |correlation|
	ZeroLag         CorrelationResult `json:"zero_lag"`         // Correlation with no lag
	LagCorrelations []LagCorrelation  `json:"lag_correlations"` // Correlations at each tested lag
	LeadingSeries   string            `json:"leading_series"`   // Which series leads (A or B), empty if no significant lag
}

// Aligns two time series by matching timestamps within a tolerance window
// Returns two slices of equal length containing only the aligned data points
func AlignDataPointsByTime(a, b []DataPoint, tolerance time.Duration) ([]DataPoint, []DataPoint, error) {
	if len(a) == 0 || len(b) == 0 {
		return nil, nil, fmt.Errorf("cannot align empty data series")
	}

	// Sort both series by timestamp
	sortedA := make([]DataPoint, len(a))
	sortedB := make([]DataPoint, len(b))
	copy(sortedA, a)
	copy(sortedB, b)
	sort.Slice(sortedA, func(i, j int) bool {
		return sortedA[i].Timestamp.Before(sortedA[j].Timestamp)
	})
	sort.Slice(sortedB, func(i, j int) bool {
		return sortedB[i].Timestamp.Before(sortedB[j].Timestamp)
	})

	alignedA := make([]DataPoint, 0)
	alignedB := make([]DataPoint, 0)

	// For each point in A, find the closest point in B within tolerance
	bIndex := 0
	for _, pointA := range sortedA {
		// Advance bIndex to find potential matches
		for bIndex < len(sortedB) && sortedB[bIndex].Timestamp.Before(pointA.Timestamp.Add(-tolerance)) {
			bIndex++
		}

		if bIndex >= len(sortedB) {
			break
		}

		// Check if the current B point is within tolerance
		timeDiff := sortedB[bIndex].Timestamp.Sub(pointA.Timestamp)
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}

		if timeDiff <= tolerance {
			alignedA = append(alignedA, pointA)
			alignedB = append(alignedB, sortedB[bIndex])
			bIndex++
		}
	}

	if len(alignedA) < 2 {
		return nil, nil, fmt.Errorf("insufficient aligned data points: got %d, need at least 2", len(alignedA))
	}

	return alignedA, alignedB, nil
}

// Calculates the Pearson correlation coefficient between two time series
// The series must be pre-aligned (same length and corresponding timestamps)
func CalculatePearsonCorrelation(a, b []DataPoint) (CorrelationResult, error) {
	if len(a) != len(b) {
		return CorrelationResult{}, fmt.Errorf("series must have equal length: got %d and %d", len(a), len(b))
	}
	if len(a) < 2 {
		return CorrelationResult{}, fmt.Errorf("need at least 2 data points for correlation: got %d", len(a))
	}

	// Extract values
	valuesA := make([]float64, len(a))
	valuesB := make([]float64, len(b))
	for i := range a {
		valuesA[i] = a[i].Value
		valuesB[i] = b[i].Value
	}

	// Calculate means
	var sumA, sumB float64
	for i := range valuesA {
		sumA += valuesA[i]
		sumB += valuesB[i]
	}
	meanA := sumA / float64(len(valuesA))
	meanB := sumB / float64(len(valuesB))

	// Calculate Pearson correlation coefficient
	var numerator, sumSqA, sumSqB float64
	for i := range valuesA {
		diffA := valuesA[i] - meanA
		diffB := valuesB[i] - meanB
		numerator += diffA * diffB
		sumSqA += diffA * diffA
		sumSqB += diffB * diffB
	}

	// Handle edge cases (constant series)
	if sumSqA == 0 || sumSqB == 0 {
		return CorrelationResult{
			Coefficient: 0,
			Strength:    CorrelationStrengthWeak,
			Direction:   CorrelationDirectionNone,
			SampleSize:  len(a),
		}, nil
	}

	coefficient := numerator / math.Sqrt(sumSqA*sumSqB)

	// Clamp to [-1, 1] to handle floating point errors
	coefficient = max(min(coefficient, 1.0), -1.0)

	return CorrelationResult{
		Coefficient: coefficient,
		Strength:    classifyCorrelationStrength(coefficient),
		Direction:   classifyCorrelationDirection(coefficient),
		SampleSize:  len(a),
	}, nil
}

// Calculates cross-correlation between two time series at various time lags
// This helps identify lead-lag relationships between metrics
func CalculateCrossCorrelation(a, b []DataPoint, maxLag time.Duration, lagStep time.Duration) (CrossCorrelationResult, error) {
	if len(a) < 2 || len(b) < 2 {
		return CrossCorrelationResult{}, fmt.Errorf("need at least 2 data points in each series")
	}

	if lagStep <= 0 {
		lagStep = time.Minute
	}

	// Calculate zero-lag correlation first
	tolerance := max(lagStep/2, 30*time.Second)

	alignedA, alignedB, err := AlignDataPointsByTime(a, b, tolerance)
	if err != nil {
		return CrossCorrelationResult{}, fmt.Errorf("failed to align series for zero-lag correlation: %w", err)
	}

	zeroLagCorr, err := CalculatePearsonCorrelation(alignedA, alignedB)
	if err != nil {
		return CrossCorrelationResult{}, fmt.Errorf("failed to calculate zero-lag correlation: %w", err)
	}

	// Calculate correlations at different lags
	lagCorrelations := make([]LagCorrelation, 0)
	lagCorrelations = append(lagCorrelations, LagCorrelation{
		Lag:         0,
		Correlation: zeroLagCorr,
	})

	maxCorr := zeroLagCorr
	optimalLag := time.Duration(0)
	leadingSeries := ""

	// Test positive lags (B leads A)
	for lag := lagStep; lag <= maxLag; lag += lagStep {
		shiftedB := shiftDataPoints(b, -lag)
		aligned1, aligned2, err := AlignDataPointsByTime(a, shiftedB, tolerance)
		if err != nil || len(aligned1) < 2 {
			continue
		}

		corr, err := CalculatePearsonCorrelation(aligned1, aligned2)
		if err != nil {
			continue
		}

		lagCorrelations = append(lagCorrelations, LagCorrelation{
			Lag:         lag,
			Correlation: corr,
		})

		if math.Abs(corr.Coefficient) > math.Abs(maxCorr.Coefficient) {
			maxCorr = corr
			optimalLag = lag
			leadingSeries = "B"
		}
	}

	// Test negative lags (A leads B)
	for lag := lagStep; lag <= maxLag; lag += lagStep {
		shiftedA := shiftDataPoints(a, -lag)
		aligned1, aligned2, err := AlignDataPointsByTime(shiftedA, b, tolerance)
		if err != nil || len(aligned1) < 2 {
			continue
		}

		corr, err := CalculatePearsonCorrelation(aligned1, aligned2)
		if err != nil {
			continue
		}

		lagCorrelations = append(lagCorrelations, LagCorrelation{
			Lag:         -lag,
			Correlation: corr,
		})

		if math.Abs(corr.Coefficient) > math.Abs(maxCorr.Coefficient) {
			maxCorr = corr
			optimalLag = -lag
			leadingSeries = "A"
		}
	}

	// Sort lag correlations by lag
	sort.Slice(lagCorrelations, func(i, j int) bool {
		return lagCorrelations[i].Lag < lagCorrelations[j].Lag
	})

	// Only report leading series if the lag is significant and correlation is moderate+
	if optimalLag == 0 || maxCorr.Strength == CorrelationStrengthWeak {
		leadingSeries = ""
	}

	return CrossCorrelationResult{
		MaxCorrelation:  maxCorr,
		OptimalLag:      optimalLag,
		ZeroLag:         zeroLagCorr,
		LagCorrelations: lagCorrelations,
		LeadingSeries:   leadingSeries,
	}, nil
}

// Shifts data points by a time offset
func shiftDataPoints(data []DataPoint, offset time.Duration) []DataPoint {
	shifted := make([]DataPoint, len(data))
	for i, dp := range data {
		shifted[i] = DataPoint{
			Value:     dp.Value,
			Timestamp: dp.Timestamp.Add(offset),
		}
	}
	return shifted
}

// Classifies correlation strength based on coefficient magnitude
func classifyCorrelationStrength(coefficient float64) CorrelationStrength {
	absCoeff := math.Abs(coefficient)
	if absCoeff >= 0.7 {
		return CorrelationStrengthStrong
	} else if absCoeff >= 0.3 {
		return CorrelationStrengthModerate
	}
	return CorrelationStrengthWeak
}

// Classifies correlation direction based on coefficient sign
func classifyCorrelationDirection(coefficient float64) CorrelationDirection {
	if math.Abs(coefficient) < 0.1 {
		return CorrelationDirectionNone
	} else if coefficient > 0 {
		return CorrelationDirectionPositive
	}
	return CorrelationDirectionNegative
}
