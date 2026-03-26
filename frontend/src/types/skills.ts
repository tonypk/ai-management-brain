export interface Skill {
  id: string
  tenant_id: string
  name: string
  category: string
  description: string
  employee_count: number
  created_at: string
}

export interface EmployeeSkill {
  id: string
  employee_id: string
  skill_id: string
  level: number // 1-5
  notes: string
  skill_name: string
  skill_category: string
  assessed_at: string
  created_at: string
}

export interface SkillMatrixEntry {
  employee_id: string
  employee_name: string
  skill_id: string
  skill_name: string
  category: string
  level: number // 0 = not assessed
}
