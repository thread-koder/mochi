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

interface RecommendationRecord {
  id: number
  workload_type: string
  workload_name: string
  namespace: string
  recommendation_mode: string
  recommendations: ContainerRecommendation[]
  status: string
  analysis_time_range?: string
  created_at: string
  updated_at?: string
  generated_at?: string
}

interface RecommendationsResponse {
  recommendations: RecommendationRecord[]
  total: number
}

export type {
  Recommendation,
  RecommendationRecord,
  ContainerRecommendation,
  RecommendationsResponse,
  RecommendationMode,
}
