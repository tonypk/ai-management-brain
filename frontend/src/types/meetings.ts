export type MeetingMood = '' | 'great' | 'good' | 'neutral' | 'concerning' | 'critical'
export type ActionItemStatus = 'open' | 'in_progress' | 'done'

export interface Meeting {
  id: string
  tenant_id: string
  employee_id: string
  manager_id: string | null
  meeting_date: string
  duration_min: number
  notes: string
  mood: MeetingMood
  follow_up: string
  employee_name: string
  manager_name?: string | null
  created_at: string
  updated_at: string
}

export interface ActionItem {
  id: string
  meeting_id: string
  title: string
  assignee_id: string | null
  status: ActionItemStatus
  due_date: string | null
  assignee_name?: string | null
  created_at: string
  updated_at: string
}

export interface OpenActionItem extends ActionItem {
  meeting_date: string
  employee_name: string
}
