interface ReplicaStatus {
  replicas: number
  ready: number
}

interface JobStatus {
  active: number
  succeeded: number
  failed: number
}

interface CronJobStatus {
  schedule: string
  suspend: boolean
}

interface PodStatus {
  phase: string
}

interface WorkloadStats {
  pods: number
  containers: number
}

interface Pod {
  name: string
  uid: string
  phase: string
  node: string
  created_at: string
}

interface Container {
  name: string
  image: string
  cpu_request: string
  cpu_limit: string
  memory_request: string
  memory_limit: string
}

type WorkloadResponse
  = | {
    type: 'Deployment' | 'StatefulSet' | 'DaemonSet'
    name: string
    created_at: string
    namespace: string
    status: ReplicaStatus
    pods: Pod[]
    containers: Container[]
    stats: WorkloadStats
  }
  | {
    type: 'Job'
    name: string
    created_at: string
    namespace: string
    status: JobStatus
    pods: Pod[]
    containers: Container[]
    stats: WorkloadStats
  }
  | {
    type: 'CronJob'
    name: string
    created_at: string
    namespace: string
    status: CronJobStatus
    pods: Pod[]
    containers: Container[]
    stats: WorkloadStats
  }
  | {
    type: 'Pod'
    name: string
    created_at: string
    namespace: string
    status: PodStatus
    pods: Pod[]
    containers: Container[]
    stats: WorkloadStats
  }

export type {
  ReplicaStatus,
  JobStatus,
  CronJobStatus,
  WorkloadResponse,
  Container,
  Pod,
}
