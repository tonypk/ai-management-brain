export interface WizardSession {
  session_id: string
  mentor_id: string
  message: string
  is_complete: boolean
}

export interface WizardAnswer {
  message: string
  is_complete: boolean
  plan?: ManagementPlan
  profile?: OrgProfile
}

export interface OrgProfile {
  industry: string
  size: number
  stage: string
  business_model?: string
  region?: string
  pain_points?: string[]
}

export interface OrgPlan {
  id: string
  industry: string
  size: number
  stage: string
  mentor_id: string
  plan: ManagementPlan
  plan_version: number
  status: string
}

export interface ManagementPlan {
  management_framework: string
  org_design: OrgDesign
  culture_principles: string[]
  policies: Record<string, unknown>
  kpi_system: KpiItem[]
  daily_questions: Record<string, string[]>
  meeting_cadence: MeetingItem[]
  alert_rules: AlertRule[]
  reasoning: string
}

export interface OrgDesign {
  philosophy: string
  structure_type: string
  units: OrgUnit[]
  support_roles?: SupportRole[]
}

export interface OrgUnit {
  name: string
  leader_type: string
  leader_role: string
  size?: number
  kpis?: string[]
}

export interface SupportRole {
  title: string
  type: string
  scope: string
}

export interface KpiItem {
  name: string
  target: string
  frequency: string
  owner: string
}

export interface MeetingItem {
  name: string
  frequency: string
  duration: string
  attendees: string
  purpose: string
}

export interface AlertRule {
  condition: string
  action: string
  message: string
}

export interface AIRole {
  id: string
  role_id: string
  title: string
  mentor_id: string
  is_active: boolean
  pending_count: number
  created_at: string
}

export type SuggestionStatus = 'pending' | 'approved' | 'rejected'

export interface AISuggestion {
  id: string
  role_id: string
  role_title: string
  capability: string
  title: string
  content: string
  status: SuggestionStatus
  created_at: string
  reviewed_at: string | null
}

export interface SetupOrgRequest {
  industry: string
  company_stage: string
  business_model: string
  team_size: number
  org_structure: string
  current_projects: string
  pain_points: string[]
  comm_tools: string[]
  culture_prefs: string
  goal_framework: string
}
