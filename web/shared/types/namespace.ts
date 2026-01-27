interface NamespaceStats {
  workloads: number
  pods: number
  containers: number
}

interface Workload {
  type: string
  name: string
  replicas: number
  ready: number
  created_at: string
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
  standalone: StandalonePod[]
  system: StandalonePod[]
}

export type { Namespace, NamespaceResponse }
