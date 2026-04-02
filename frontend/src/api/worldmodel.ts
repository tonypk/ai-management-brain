import { get } from './client'

export interface WorldModelOverview {
  skill_count: number
  relationship_count: number
  active_blocker_count: number
  growth_events_month: number
  blocker_breakdown: { category: string; count: number }[]
}

export interface SkillRow {
  employee_name: string
  skill_name: string
  proficiency: string
  confidence: number
  mention_count: number
}

export interface RelationshipRow {
  employee_a_name: string
  employee_b_name: string
  relation_type: string
  context: string
  strength: number
  interaction_count: number
}

export interface BlockerRow {
  employee_name: string
  category: string
  description: string
  status: string
  recurrence_count: number
}

export interface InsightRow {
  dimension: string
  insight_text: string
  confidence: number
  generated_at: string
}

export interface EmployeeWorldModel {
  skills: SkillRow[]
  relationships: RelationshipRow[]
  blockers: BlockerRow[]
  growth_events: { event_type: string; description: string; detected_at: string }[]
}

export async function getWorldModelOverview(): Promise<WorldModelOverview> {
  const res = await get<{ data: WorldModelOverview }>('/world-model/overview')
  return res.data
}

export async function getWorldModelSkills(): Promise<SkillRow[]> {
  const res = await get<{ data: SkillRow[] }>('/world-model/skills')
  return res.data ?? []
}

export async function getWorldModelRelationships(): Promise<RelationshipRow[]> {
  const res = await get<{ data: RelationshipRow[] }>('/world-model/relationships')
  return res.data ?? []
}

export async function getWorldModelBlockers(): Promise<BlockerRow[]> {
  const res = await get<{ data: BlockerRow[] }>('/world-model/blockers')
  return res.data ?? []
}

export async function getWorldModelInsights(): Promise<InsightRow[]> {
  const res = await get<{ data: InsightRow[] }>('/world-model/insights')
  return res.data ?? []
}

export async function getEmployeeWorldModel(employeeId: string): Promise<EmployeeWorldModel> {
  const res = await get<{ data: EmployeeWorldModel }>(`/employees/${employeeId}/world-model`)
  return res.data
}
