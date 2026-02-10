type Direction = 'stable' | 'increasing' | 'decreasing'

type Severity = 'low' | 'medium' | 'high'

interface DataPoint {
  value: number
  timestamp: string
}

interface TimeSeries {
  cpu: DataPoint[]
  memory: DataPoint[]
}

interface StabilityResult {
  cpu_throttling: number
  cpu_pressure: number
  memory_fail_cnt: number
  memory_oom: number
  memory_pressure: number
  restarts: number
  stability_score: number
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

interface CPUUtilization {
  current: number
  stats: StatsResult
  trend: TrendResult
  anomalies: AnomalyResult
}

interface MemoryUtilization {
  current: number
  stats: StatsResult
  trend: TrendResult
  anomalies: AnomalyResult
}

interface UtilizationResult {
  cpu: CPUUtilization
  memory: MemoryUtilization
}

interface CPUProvisioning {
  request_utilization: number
  limit_utilization: number
  current_request?: number | null
  current_limit?: number | null
  is_over_provisioned: boolean
  is_under_provisioned: boolean
  efficiency: number
  confidence: number
}

interface MemoryProvisioning {
  request_utilization: number
  limit_utilization: number
  current_request?: number | null
  current_limit?: number | null
  is_over_provisioned: boolean
  is_under_provisioned: boolean
  efficiency: number
  confidence: number
}

interface ProvisioningResult {
  cpu: CPUProvisioning
  memory: MemoryProvisioning
  efficiency: number
}

interface ContainerAnalysis {
  container_name: string
  utilization: UtilizationResult
  stability: StabilityResult
  provisioning: ProvisioningResult
  time_series?: TimeSeries
}

interface PodAnalysis {
  pod_uid: string
  pod_name: string
  containers: ContainerAnalysis[]
  utilization: UtilizationResult
  stability: StabilityResult
  time_series?: TimeSeries
}

interface WorkloadAnalysis {
  workload_type: string
  workload_name: string
  namespace: string
  pods: PodAnalysis[]
  utilization: UtilizationResult
  stability: StabilityResult
  time_series?: TimeSeries
}

interface NamespaceAnalysis {
  namespace: string
  utilization: UtilizationResult
  stability: StabilityResult
  time_series?: TimeSeries
  workloads?: WorkloadAnalysis[]
}

type WorkloadContainerAnalysis = WorkloadAnalysis['pods'][number]['containers'][number]
type ProvisioningType = WorkloadContainerAnalysis['provisioning']['cpu']

type RecommendationMode = 'cost_optimized' | 'burstable' | 'guaranteed'

interface CPURecommendation {
  current_request?: string | null
  recommended_request?: string | null
  request_change_percent?: number | null
  current_limit?: string | null
  recommended_limit?: string | null
  limit_change_percent?: number | null
}

interface MemoryRecommendation {
  current_request?: string | null
  recommended_request?: string | null
  request_change_percent?: number | null
  current_limit?: string | null
  recommended_limit?: string | null
  limit_change_percent?: number | null
}

interface ContainerRecommendation {
  container_name: string
  cpu: CPURecommendation
  memory: MemoryRecommendation
  confidence_score: number
}

interface Recommendation {
  workload_type: string
  workload_name: string
  namespace: string
  recommendation_mode: RecommendationMode
  recommendations: ContainerRecommendation[]
  analysis_time_range: string
}

export type {
  UtilizationResult,
  NamespaceAnalysis,
  WorkloadAnalysis,
  WorkloadContainerAnalysis,
  ProvisioningType,
  StabilityResult,
  Recommendation,
}
