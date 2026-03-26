export interface Project {
  id: string
  tenant_id: string
  name: string
  description: string | null
  owner_id: string | null
  owner_team_id: string | null
  owner_name?: string
  team_name?: string
  status: string
  priority: string
  linked_goal_ids: string[]
  linked_metric_ids: string[]
  blockers: string[]
  source_system: string | null
  source_ref: string | null
  start_date: string | null
  due_date: string | null
  created_at: string
  updated_at: string
}
