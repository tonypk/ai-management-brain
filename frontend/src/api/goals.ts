import { get, post, put, del } from './client'
import type { Objective, KeyResult, GoalSnapshot } from '@/types'

// Backend returns goals with key_results_json as a JSON column.
// We map it to match the Objective interface.
interface GoalRow {
  id: string
  tenant_id: string
  owner_id: string | null
  title: string
  description: string
  status: string
  cycle: string
  created_at: string
  updated_at: string
  key_results_json: KeyResult[] | null
}

function mapGoalRow(row: GoalRow): Objective {
  return {
    id: row.id,
    title: row.title,
    description: row.description,
    status: row.status as Objective['status'],
    cycle: row.cycle,
    owner_id: row.owner_id,
    key_results: row.key_results_json ?? [],
    created_at: row.created_at,
    updated_at: row.updated_at,
  }
}

export async function listGoals(cycle: string): Promise<Objective[]> {
  const res = await get<{ data: GoalRow[] }>(`/goals?cycle=${encodeURIComponent(cycle)}`)
  return res.data.map(mapGoalRow)
}

export async function createGoal(data: {
  title: string
  description: string
  cycle: string
  owner_id?: string | null
  status?: string
}): Promise<Objective> {
  const res = await post<{ data: GoalRow }>('/goals', data)
  return { ...mapGoalRow(res.data), key_results: [] }
}

export async function updateGoal(id: string, data: {
  title: string
  description: string
  cycle: string
  owner_id?: string | null
  status: string
}): Promise<void> {
  await put<{ data: unknown }>(`/goals/${id}`, data)
}

export async function deleteGoal(id: string): Promise<void> {
  await del<{ data: unknown }>(`/goals/${id}`)
}

export async function createKeyResult(goalId: string, data: {
  title: string
  target: number
  current_value?: number
  unit?: string
  due_date?: string | null
}): Promise<KeyResult> {
  const res = await post<{ data: KeyResult }>(`/goals/${goalId}/key-results`, data)
  return res.data
}

export async function updateKeyResult(goalId: string, krId: string, data: {
  title: string
  target: number
  current_value: number
  unit?: string
  due_date?: string | null
}): Promise<void> {
  await put<{ data: unknown }>(`/goals/${goalId}/key-results/${krId}`, data)
}

export async function deleteKeyResult(goalId: string, krId: string): Promise<void> {
  await del<{ data: unknown }>(`/goals/${goalId}/key-results/${krId}`)
}

export async function listSnapshots(goalId: string): Promise<GoalSnapshot[]> {
  const res = await get<{ data: GoalSnapshot[] }>(`/goals/${goalId}/snapshots`)
  return res.data
}
