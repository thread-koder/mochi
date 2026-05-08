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

interface WorkloadResponse {
  namespace: string
  type: string
  name: string
  replicas: number
  ready: number
  created_at: string
  pods: Pod[]
  containers: Container[]
  stats: WorkloadStats
}

export type { WorkloadResponse, Container, Pod }
