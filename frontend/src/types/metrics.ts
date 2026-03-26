export interface Metric {
  id: string
  tenant_id: string
  name: string
  display_name: string
  formula: string
  unit: string | null
  source: string
  refresh_frequency: string | null
  target_value: string | null
  alert_threshold: string | null
  owner_id: string | null
  owner_team_id: string | null
  tags: string[]
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface MetricWithValue extends Metric {
  owner_name?: string
  latest_value: string | null
  latest_observed_at: string | null
}

export interface MetricValue {
  id: number
  metric_id: string
  observed_at: string
  value: string
  dimensions: Record<string, unknown>
  source_ref: string | null
  created_at: string
}
