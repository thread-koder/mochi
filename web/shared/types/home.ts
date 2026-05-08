interface Stats {
  namespaces: number
  workloads: number
  pods: number
  health_score: number
}

interface Activity {
  type: string
  message: string
  timestamp: string
}

interface HomeResponse {
  cluster_name: string
  stats: Stats
  health_checks: Record<string, boolean>
  activities: Activity[]
}

export type { HomeResponse, Activity }
