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

interface ResourceUtilization {
  current: number
  stats: StatsResult
  trend: TrendResult
  anomalies: AnomalyResult
  sample_size: number
}

interface UtilizationResult {
  cpu: ResourceUtilization
  memory: ResourceUtilization
}

interface ResourceProvisioning {
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
  cpu: ResourceProvisioning
  memory: ResourceProvisioning
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

export type {
  UtilizationResult,
  NamespaceAnalysis,
  WorkloadAnalysis,
  ContainerAnalysis,
  ResourceProvisioning,
  StabilityResult,
  PodAnalysis,
}
