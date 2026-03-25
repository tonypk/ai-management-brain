export interface Employee {
  id: string
  name: string
  culture_code: string
  role: string
  is_active: boolean
  has_telegram: boolean
  invite_code: string
  job_title: string
  responsibilities: string
  country: string
  language: string
}

export interface EmployeeWithChannels {
  id: string
  name: string
  telegram_id: boolean
  signal_phone: string
  slack_id: string
  lark_id: string
  preferred_channel: string
  culture_code: string
  role: string
}
