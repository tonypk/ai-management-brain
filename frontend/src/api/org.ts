import { get, put, post } from './client'
import type { OrgPlan, ManagementPlan, AIRole, AISuggestion, SetupOrgRequest } from '@/types'

export async function setupOrg(data: SetupOrgRequest): Promise<OrgPlan> {
  const res = await post<{ data: OrgPlan }>('/org/setup', data)
  return res.data
}

export async function getOrgPlan(): Promise<OrgPlan> {
  const res = await get<{ data: OrgPlan }>('/org/plan')
  return res.data
}

export async function adjustPlan(feedback: string): Promise<{ plan: ManagementPlan; plan_version: number }> {
  const res = await put<{ data: { plan: ManagementPlan; plan_version: number } }>('/org/plan', { feedback })
  return res.data
}

export async function activatePlan(): Promise<{ status: string; roles_activated: number }> {
  const res = await post<{ data: { status: string; roles_activated: number } }>('/org/plan/activate')
  return res.data
}

export async function getOrgRoles(): Promise<AIRole[]> {
  const res = await get<{ data: AIRole[] }>('/org/roles')
  return res.data
}

export async function getOrgSuggestions(): Promise<AISuggestion[]> {
  const res = await get<{ data: AISuggestion[] }>('/org/suggestions')
  return res.data
}

export async function approveSuggestion(id: string): Promise<void> {
  await post<{ data: unknown }>(`/org/suggestions/${id}/approve`)
}

export async function rejectSuggestion(id: string): Promise<void> {
  await post<{ data: unknown }>(`/org/suggestions/${id}/reject`)
}
