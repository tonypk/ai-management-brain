import { get, post, put, del } from './client'
import type { IncentiveRule, IncentiveScore } from '@/types'

export async function listIncentiveRules(): Promise<IncentiveRule[]> {
  const res = await get<{ data: IncentiveRule[] }>('/incentives/rules')
  return res.data
}

export async function createIncentiveRule(data: {
  name: string
  reward_model?: string
  payout_cycle?: string
  attribution_rules?: Record<string, unknown>
  penalty_rules?: unknown[]
  scoring_formula?: Record<string, unknown>
  applies_to?: unknown[]
}): Promise<IncentiveRule> {
  const res = await post<{ data: IncentiveRule }>('/incentives/rules', data)
  return res.data
}

export async function updateIncentiveRule(
  id: string,
  data: {
    name: string
    reward_model?: string
    payout_cycle?: string
    attribution_rules?: Record<string, unknown>
    penalty_rules?: unknown[]
    scoring_formula?: Record<string, unknown>
    applies_to?: unknown[]
    is_active?: boolean
  },
): Promise<IncentiveRule> {
  const res = await put<{ data: IncentiveRule }>(`/incentives/rules/${id}`, data)
  return res.data
}

export async function deleteIncentiveRule(id: string): Promise<void> {
  await del(`/incentives/rules/${id}`)
}

export async function listIncentiveScores(period: string): Promise<IncentiveScore[]> {
  const res = await get<{ data: IncentiveScore[] }>(`/incentives/scores?period=${encodeURIComponent(period)}`)
  return res.data
}
