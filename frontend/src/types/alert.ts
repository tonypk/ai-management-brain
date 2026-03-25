export interface Alert {
  employee_id: string
  employee_name: string
  alert_type: string
  severity: 'warning' | 'critical'
  message: string
  consecutive_misses?: number
  last_checkin?: string
  chase_count?: number
}

export interface CheckinStatus {
  date: string
  total_employees: number
  submitted: SubmittedEmployee[]
  pending: PendingEmployee[]
  missed: string[]
}

export interface SubmittedEmployee {
  id: string
  name: string
  submitted_at: string
}

export interface PendingEmployee {
  id: string
  name: string
  chase_count: number
}
