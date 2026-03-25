export interface Tenant {
  id: string
  name: string
  timezone: string
  mentor_id: string
  mentor_blend: import('./mentor').BlendConfig | null
}

export interface BillingStatus {
  plan: string
  status: string
  employee_limit: number
  employee_count: number
  features: string[]
  billing_cycle?: string
  next_billing_date?: string
  amount?: number
}
