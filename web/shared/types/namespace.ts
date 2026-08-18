interface NamespaceStats {
  workloads: number
  pods: number
  containers: number
}

type Workload
  = | {
    type: 'Deployment' | 'StatefulSet' | 'DaemonSet'
    name: string
    created_at: string
    status: ReplicaStatus
  }
  | {
    type: 'Job'
    name: string
    created_at: string
    status: JobStatus
  }
  | {
    type: 'CronJob'
    name: string
    created_at: string
    status: CronJobStatus
  }

interface StandalonePod {
  name: string
  phase: string
  node: string
  created_at: string
}

interface Namespace {
  name: string
  phase: string
}

interface NamespaceResponse {
  name: string
  phase: string
  stats: NamespaceStats
  workloads: Workload[]
  standalone_pods: StandalonePod[]
  system_pods: StandalonePod[]
}

export type { Namespace, NamespaceResponse, StandalonePod, Workload }
