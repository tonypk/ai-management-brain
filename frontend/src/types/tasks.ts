export interface Task {
  id: string
  tenant_id: string
  project_id: string | null
  goal_id: string | null
  key_result_id: string | null
  title: string
  description: string | null
  owner_id: string | null
  owner_team_id: string | null
  owner_name?: string
  project_name?: string
  status: string
  priority: string
  due_at: string | null
  source_system: string | null
  source_ref: string | null
  created_by_agent: boolean
  created_at: string
  updated_at: string
}

export interface TaskStats {
  status: string
  count: number
}
