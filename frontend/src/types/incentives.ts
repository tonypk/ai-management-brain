export interface IncentiveRule {
  id: string
  tenant_id: string
  name: string
  reward_model: string
  payout_cycle: string
  attribution_rules: Record<string, unknown>
  penalty_rules: unknown[]
  scoring_formula: Record<string, unknown>
  applies_to: unknown[]
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface IncentiveScore {
  id: string
  tenant_id: string
  rule_id: string
  person_id: string
  person_name?: string
  rule_name?: string
  period: string
  score: string
  score_breakdown: Record<string, unknown>
  payout_weight: string
  attribution_confidence: string
  status: string
  calculated_at: string
  reviewed_at: string | null
}
