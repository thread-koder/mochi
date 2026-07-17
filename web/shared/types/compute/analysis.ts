import type {
  DataPoint,
  StatsResult,
  TrendResult,
  AnomalyResult,
} from '#shared/types/timeseries'

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

interface CPUUtilization {
  current: number
  stats: StatsResult
  trend: TrendResult
  anomalies: AnomalyResult
  sample_size: number
}

interface MemoryUtilization {
  current: number
  stats: StatsResult
  trend: TrendResult
  anomalies: AnomalyResult
  sample_size: number
}

interface UtilizationResult {
  cpu: CPUUtilization
  memory: MemoryUtilization
}

interface CPUProvisioning {
  request_utilization: number
  limit_utilization: number
  current_request: number | null
  current_limit: number | null
  is_over_provisioned: boolean
  is_under_provisioned: boolean
  efficiency: number
  confidence: number
}

interface MemoryProvisioning {
  request_utilization: number
  limit_utilization: number
  current_request: number | null
  current_limit: number | null
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
  workloads: WorkloadAnalysis[]
}

type ResourceProvisioning = CPUProvisioning | MemoryProvisioning

export type {
  UtilizationResult,
  NamespaceAnalysis,
  WorkloadAnalysis,
  ContainerAnalysis,
  ResourceProvisioning,
  StabilityResult,
  PodAnalysis,
}
