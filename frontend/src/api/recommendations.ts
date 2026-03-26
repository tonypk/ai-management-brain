import { get, post, del } from './client'
import type { Recommendation, RecommendationSummary, ActionResult, ExecuteAllResult } from '@/types'

export async function listRecommendations(status = '', category = ''): Promise<Recommendation[]> {
  const params = new URLSearchParams()
  if (status) params.set('status', status)
  if (category) params.set('category', category)
  const qs = params.toString()
  const res = await get<{ data: Recommendation[] }>(`/recommendations${qs ? `?${qs}` : ''}`)
  return res.data ?? []
}

export async function getRecommendationSummary(): Promise<RecommendationSummary> {
  const res = await get<{ data: RecommendationSummary }>('/recommendations/summary')
  return res.data
}

export async function executeAction(id: string, actionIndex: number): Promise<ActionResult> {
  const res = await post<{ data: ActionResult }>(`/recommendations/${id}/execute`, { action_index: actionIndex })
  return res.data
}

export async function executeAll(id: string): Promise<ExecuteAllResult> {
  const res = await post<{ data: ExecuteAllResult }>(`/recommendations/${id}/execute-all`, {})
  return res.data
}

export async function dismissRecommendation(id: string): Promise<void> {
  await post<{ data: unknown }>(`/recommendations/${id}/dismiss`, {})
}

export async function deleteRecommendation(id: string): Promise<void> {
  await del<{ data: unknown }>(`/recommendations/${id}`)
}
