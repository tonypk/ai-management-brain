export interface DashboardStats {
  employee_count: number
  today_submissions: number
  current_mentor: string
  last_summary_date: string
}

export interface AnalyticsOverview {
  today: {
    date: string
    reports: number
    employees: number
    submission_rate: number
  }
  trend_7d: TrendPoint[]
  sentiment: Record<string, number>
  health_score: number
}

export interface TrendPoint {
  date: string
  count: number
  rate: number
}

export interface EmployeeActivity {
  id: string
  name: string
  submitted_7d: number
  missed_7d: number
  last_sentiment: string
  culture_code: string
}
