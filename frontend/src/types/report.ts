export interface Report {
  id: string
  employee_id: string
  employee_name: string
  report_date: string
  answers: Record<string, string>
  submitted_at: string
  blockers?: string
  sentiment?: string
}

export interface Summary {
  id: string
  summary_date: string
  content: string
  submission_rate: number
  blockers_count: number
  key_metrics: Record<string, unknown>
}

export interface AdminReport {
  id: string
  employee_id: string
  employee_name: string
  report_date: string
  answers: Record<string, string>
  blockers?: string
  sentiment?: string
  channel: string
  submitted_at: string
}
