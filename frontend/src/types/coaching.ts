export type RiskLevel = 'high' | 'medium' | 'low' | 'none'

export interface AtRiskEmployee {
  id: string
  name: string
  risk: RiskLevel
  missed_7d: number
  last_sentiment: string
  culture_code: string
}

export interface TalkingPoint {
  employee_name: string
  priority: RiskLevel
  points: string[]
}

export interface CoachingMessage {
  role: 'user' | 'assistant'
  content: string
}
