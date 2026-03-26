export interface CommunicationEvent {
  id: string
  tenant_id: string
  source_type: string
  source_id: string | null
  platform: string
  event_type: string
  actor_id: string | null
  target_id: string | null
  actor_name?: string
  target_name?: string
  related_task_id: string | null
  related_project_id: string | null
  related_goal_id: string | null
  payload: Record<string, unknown>
  confidence: string
  occurred_at: string
  created_at: string
}

export interface ExecutionSignal {
  id: string
  tenant_id: string
  subject_type: string
  subject_id: string
  signal_type: string
  score: string
  reasons: string[]
  time_window: string
  generated_at: string
}

export interface WorkingMemorySnapshot {
  id: string
  tenant_id: string
  snapshot_type: string
  content: Record<string, unknown>
  generated_by: string
  generated_at: string
}

export interface CompanyState {
  top_risks: ExecutionSignal[]
  overdue_tasks: unknown[]
  task_stats: { status: string; count: number }[]
  event_counts: { event_type: string; count: number }[]
  blocked_projects: unknown[]
  working_memory: WorkingMemorySnapshot | null
}

export interface Workflow {
  id: string
  tenant_id: string
  name: string
  category: string | null
  trigger_conditions: Record<string, unknown>
  steps: unknown[]
  approval_rules: Record<string, unknown>
  escalation_rules: Record<string, unknown>
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface ReportingLine {
  id: string
  tenant_id: string
  manager_id: string
  report_id: string
  manager_name?: string
  report_name?: string
  relationship_type: string
  created_at: string
}
