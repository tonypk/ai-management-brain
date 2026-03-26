export type RecommendationCategory = 'people' | 'project' | 'kpi' | 'organization'
export type RecommendationPriority = 'critical' | 'high' | 'medium' | 'low'
export type RecommendationStatus = 'pending' | 'accepted' | 'dismissed' | 'executed' | 'expired'

export interface SuggestedAction {
  type: string
  params: Record<string, unknown>
  label: string
}

export interface EvidenceSignal {
  name: string
  value: number
}

export interface EvidenceEmployee {
  name: string
  issue: string
}

export interface EvidenceMetric {
  name: string
  trend: string
}

export interface EvidenceTask {
  id: string
  issue: string
}

export interface Evidence {
  signals?: EvidenceSignal[]
  employees?: EvidenceEmployee[]
  metrics?: EvidenceMetric[]
  tasks?: EvidenceTask[]
}

export interface Recommendation {
  id: string
  tenant_id: string
  category: RecommendationCategory
  priority: RecommendationPriority
  title: string
  description: string
  suggested_actions: SuggestedAction[]
  evidence: Evidence
  source: 'daily_scan' | 'realtime_trigger'
  status: RecommendationStatus
  target_entity_type: string | null
  target_entity_id: string | null
  expires_at: string
  created_at: string
  reviewed_at: string | null
  executed_at: string | null
}

export interface RecommendationSummary {
  pending_count: number
  top: Recommendation[]
}

export interface ActionResult {
  index: number
  success: boolean
  message?: string
  error?: string
  skipped?: string
  needs_confirmation?: boolean
  link?: string
}

export interface ExecuteAllResult {
  results: ActionResult[]
  all_done: boolean
}
