package timeseries

import "time"

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
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// Represents percentile calculation results
type PercentileResult struct {
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

// Represents trend analysis results
type TrendResult struct {
	Direction Direction `json:"direction"` // "increasing", "decreasing", or "stable"
	Slope     float64   `json:"slope"`     // Linear regression slope
	Strength  float64   `json:"strength"`  // Correlation coefficient (0-1)
}

// Represents statistical calculation results
type StatsResult struct {
	Mean       float64          `json:"mean"`
	Median     float64          `json:"median"`
	StdDev     float64          `json:"std_dev"`
	Min        float64          `json:"min"`
	Max        float64          `json:"max"`
	Percentile PercentileResult `json:"percentile"`
}

// Represents anomaly detection results
type AnomalyResult struct {
	Anomalies    []Anomaly `json:"anomalies"`
	AnomalyCount int       `json:"anomaly_count"`
	Threshold    float64   `json:"threshold"`
}

// Represents a detected anomaly
type Anomaly struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Index     int       `json:"index"`
	Deviation float64   `json:"deviation"` // How many standard deviations from mean
	Severity  Severity  `json:"severity"`  // "low", "medium", "high"
}
