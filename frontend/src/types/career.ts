export interface CareerLevel {
  id: string
  tenant_id: string
  title: string
  level_order: number
  description: string
  requirements: string
  created_at: string
}

export interface CareerPath {
  id: string
  employee_id: string
  current_level_id: string | null
  target_level_id: string | null
  target_date: string | null
  notes: string
  employee_name: string
  current_level_title: string | null
  current_level_order: number | null
  target_level_title: string | null
  target_level_order: number | null
  created_at: string
  updated_at: string
}
