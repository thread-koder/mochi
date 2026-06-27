package timeseries

import (
	"fmt"
	"math"
	"sort"
	"time"
)

type CorrelationStrength string

const (
	CorrelationStrengthWeak     CorrelationStrength = "weak"
	CorrelationStrengthModerate CorrelationStrength = "moderate"
	CorrelationStrengthStrong   CorrelationStrength = "strong"
)

type CorrelationDirection string

const (
	CorrelationDirectionPositive CorrelationDirection = "positive"
	CorrelationDirectionNegative CorrelationDirection = "negative"
	CorrelationDirectionNone     CorrelationDirection = "none"
)

type CorrelationResult struct {
	Coefficient float64              `json:"coefficient"`
	Strength    CorrelationStrength  `json:"strength"`
	Direction   CorrelationDirection `json:"direction"`
	SampleSize  int                  `json:"sample_size"`
}

type LagCorrelation struct {
	Lag         time.Duration     `json:"lag"`
	Correlation CorrelationResult `json:"correlation"`
}

type CrossCorrelationResult struct {
	MaxCorrelation  CorrelationResult `json:"max_correlation"`
	OptimalLag      time.Duration     `json:"optimal_lag"`
	ZeroLag         CorrelationResult `json:"zero_lag"`
	LagCorrelations []LagCorrelation  `json:"lag_correlations"`
	LeadingSeries   string            `json:"leading_series"`
}

// AlignDataPointsByTime pairs samples whose timestamps fall within tolerance.
// This accepts small scrape/sampling skew without forcing exact timestamp equality.
func AlignDataPointsByTime(a, b []DataPoint, tolerance time.Duration) ([]DataPoint, []DataPoint, error) {
	if len(a) == 0 || len(b) == 0 {
		return nil, nil, fmt.Errorf("cannot align empty data series")
	}

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

	bIndex := 0
	for _, pointA := range sortedA {
		for bIndex < len(sortedB) && sortedB[bIndex].Timestamp.Before(pointA.Timestamp.Add(-tolerance)) {
			bIndex++
		}

		if bIndex >= len(sortedB) {
			break
		}

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

func CalculatePearsonCorrelation(a, b []DataPoint) (CorrelationResult, error) {
	if len(a) != len(b) {
		return CorrelationResult{}, fmt.Errorf("series must have equal length: got %d and %d", len(a), len(b))
	}
	if len(a) < 2 {
		return CorrelationResult{}, fmt.Errorf("need at least 2 data points for correlation: got %d", len(a))
	}

	valuesA := make([]float64, len(a))
	valuesB := make([]float64, len(b))
	for i := range a {
		valuesA[i] = a[i].Value
		valuesB[i] = b[i].Value
	}

	var sumA, sumB float64
	for i := range valuesA {
		sumA += valuesA[i]
		sumB += valuesB[i]
	}
	meanA := sumA / float64(len(valuesA))
	meanB := sumB / float64(len(valuesB))

	var numerator, sumSqA, sumSqB float64
	for i := range valuesA {
		diffA := valuesA[i] - meanA
		diffB := valuesB[i] - meanB
		numerator += diffA * diffB
		sumSqA += diffA * diffA
		sumSqB += diffB * diffB
	}

	// Constant series have zero variance, so correlation is undefined in practice.
	// We report neutral correlation to keep API output stable for callers.
	if sumSqA == 0 || sumSqB == 0 {
		return CorrelationResult{
			Coefficient: 0,
			Strength:    CorrelationStrengthWeak,
			Direction:   CorrelationDirectionNone,
			SampleSize:  len(a),
		}, nil
	}

	coefficient := numerator / math.Sqrt(sumSqA*sumSqB)

	// Guard against floating-point drift slightly outside the valid Pearson range.
	coefficient = max(min(coefficient, 1.0), -1.0)

	return CorrelationResult{
		Coefficient: coefficient,
		Strength:    classifyCorrelationStrength(coefficient),
		Direction:   classifyCorrelationDirection(coefficient),
		SampleSize:  len(a),
	}, nil
}

// CalculateCrossCorrelation scans lag offsets to detect lead/lag relationships.
func CalculateCrossCorrelation(a, b []DataPoint, maxLag time.Duration, lagStep time.Duration) (CrossCorrelationResult, error) {
	if len(a) < 2 || len(b) < 2 {
		return CrossCorrelationResult{}, fmt.Errorf("need at least 2 data points in each series")
	}

	if lagStep <= 0 {
		lagStep = time.Minute
	}

	tolerance := max(lagStep/2, 30*time.Second)

	alignedA, alignedB, err := AlignDataPointsByTime(a, b, tolerance)
	if err != nil {
		return CrossCorrelationResult{}, err
	}

	zeroLagCorr, err := CalculatePearsonCorrelation(alignedA, alignedB)
	if err != nil {
		return CrossCorrelationResult{}, err
	}

	lagCorrelations := make([]LagCorrelation, 0)
	lagCorrelations = append(lagCorrelations, LagCorrelation{
		Lag:         0,
		Correlation: zeroLagCorr,
	})

	maxCorr := zeroLagCorr
	optimalLag := time.Duration(0)
	leadingSeries := ""

	// Positive lag means B is shifted earlier, so B potentially leads A.
	for lag := lagStep; lag <= maxLag; lag += lagStep {
		shiftedB := shiftDataPoints(b, -lag)
		alignedA, alignedB, err := AlignDataPointsByTime(a, shiftedB, tolerance)
		if err != nil || len(alignedA) < 2 {
			continue
		}

		corr, err := CalculatePearsonCorrelation(alignedA, alignedB)
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

	// Negative lag means A is shifted earlier, so A potentially leads B.
	for lag := lagStep; lag <= maxLag; lag += lagStep {
		shiftedA := shiftDataPoints(a, -lag)
		alignedA, alignedB, err := AlignDataPointsByTime(shiftedA, b, tolerance)
		if err != nil || len(alignedA) < 2 {
			continue
		}

		corr, err := CalculatePearsonCorrelation(alignedA, alignedB)
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

	sort.Slice(lagCorrelations, func(i, j int) bool {
		return lagCorrelations[i].Lag < lagCorrelations[j].Lag
	})

	// Avoid overclaiming leadership when the best signal is weak or at zero lag.
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

func classifyCorrelationStrength(coefficient float64) CorrelationStrength {
	absCoeff := math.Abs(coefficient)
	if absCoeff >= 0.7 {
		return CorrelationStrengthStrong
	} else if absCoeff >= 0.3 {
		return CorrelationStrengthModerate
	}
	return CorrelationStrengthWeak
}

func classifyCorrelationDirection(coefficient float64) CorrelationDirection {
	if math.Abs(coefficient) < 0.1 {
		return CorrelationDirectionNone
	} else if coefficient > 0 {
		return CorrelationDirectionPositive
	}
	return CorrelationDirectionNegative
}
