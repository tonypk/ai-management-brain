export interface InsightRecord {
  id: string
  content: string          // AI response (markdown)
  context_summary: string  // One-line data summary at generation time
  health_score: number
  alert_count: number
  created_at: string
}

export type DigestPeriod = 'weekly' | 'monthly'

export interface DigestRecord {
  id: string
  period: DigestPeriod
  period_label: string     // "Week of Mar 24, 2026" / "March 2026"
  content: string          // AI response (markdown)
  created_at: string
}

export interface InsightsStorage {
  meta: { version: 1; updated_at: string }
  insights: InsightRecord[]
}

export interface DigestsStorage {
  meta: { version: 1; updated_at: string }
  digests: DigestRecord[]
}
