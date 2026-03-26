export type ReviewCycleStatus = 'draft' | 'active' | 'completed'
export type ReviewStatus = 'pending' | 'in_progress' | 'submitted' | 'acknowledged'

export interface ReviewCycle {
  id: string
  tenant_id: string
  title: string
  period: string
  status: ReviewCycleStatus
  start_date: string
  end_date: string
  created_at: string
  updated_at: string
}

export interface PerformanceReview {
  id: string
  cycle_id: string
  employee_id: string
  reviewer_id: string | null
  status: ReviewStatus
  self_rating: number | null
  manager_rating: number | null
  self_summary: string
  manager_summary: string
  strengths: string
  improvements: string
  submitted_at: string | null
  acknowledged_at: string | null
  employee_name: string
  reviewer_name?: string | null
  created_at: string
  updated_at: string
}
