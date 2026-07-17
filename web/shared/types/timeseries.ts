type Direction = 'stable' | 'increasing' | 'decreasing'

type Severity = 'low' | 'medium' | 'high'

interface DataPoint {
  value: number
  timestamp: string
}

interface PercentileResult {
  p50: number
  p95: number
  p99: number
}

interface StatsResult {
  mean: number
  median: number
  std_dev: number
  min: number
  max: number
  percentile: PercentileResult
}

interface TrendResult {
  direction: Direction
  slope: number
  strength: number
}

interface Anomaly {
  value: number
  timestamp: string
  index: number
  deviation: number
  severity: Severity
}

interface AnomalyResult {
  anomalies: Anomaly[]
  anomaly_count: number
  threshold: number
}

export type {
  DataPoint,
  StatsResult,
  TrendResult,
  AnomalyResult,
}
