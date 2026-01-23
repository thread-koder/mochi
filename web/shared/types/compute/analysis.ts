type Direction = 'stable' | 'increasing' | 'decreasing'

type Severity = 'low' | 'medium' | 'high'

interface DataPoint {
  value: number
  timestamp: string
}

interface ResourceMetrics {
  cpu: DataPoint[]
  memory: DataPoint[]
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
  is_over_provisioned: boolean
  is_under_provisioned: boolean
  efficiency: number
  confidence: number
}

interface MemoryProvisioning {
  request_utilization: number
  limit_utilization: number
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
  provisioning: ProvisioningResult
  time_series?: ResourceMetrics
}

interface PodAnalysis {
  pod_uid: string
  pod_name: string
  containers: ContainerAnalysis[]
  utilization: UtilizationResult
  time_series?: ResourceMetrics
}

interface WorkloadAnalysis {
  workload_type: string
  workload_name: string
  namespace: string
  pods: PodAnalysis[]
  utilization: UtilizationResult
  time_series?: ResourceMetrics
}

interface NamespaceAnalysis {
  namespace: string
  utilization: UtilizationResult
  time_series?: ResourceMetrics
  workloads?: WorkloadAnalysis[]
}

type WorkloadContainerAnalysis = WorkloadAnalysis['pods'][number]['containers'][number]

export type { UtilizationResult, NamespaceAnalysis, WorkloadAnalysis, WorkloadContainerAnalysis }
