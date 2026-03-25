import { getDashboard, getAnalyticsOverview, getEmployeeActivity } from '@/api/dashboard'
import { getAlerts } from '@/api/alerts'
import { usePlanningStore } from '@/stores/planning'
import type {
  DashboardStats, AnalyticsOverview, EmployeeActivity, Alert,
  Objective, BoardRecord,
} from '@/types'

export interface CollectedData {
  dashboard: DashboardStats | null
  analytics: AnalyticsOverview | null
  activity: EmployeeActivity[]
  alerts: Alert[]
  goals: Objective[]
  boardRecords: BoardRecord[]
}

export function useDataCollector() {
  async function collect(): Promise<CollectedData> {
    const store = usePlanningStore()

    const [dashboard, analytics, activity, alerts] = await Promise.all([
      getDashboard().catch(() => null),
      getAnalyticsOverview().catch(() => null),
      getEmployeeActivity().catch(() => []),
      getAlerts().catch(() => []),
    ])

    const goals = [...store.objectives]
    const boardRecords = store.boardRecords.slice(0, 5)

    return { dashboard, analytics, activity, alerts, goals, boardRecords }
  }

  function formatContext(data: CollectedData): string {
    const lines: string[] = []

    // Team Data
    lines.push('=== TEAM DATA ===')
    const hs = data.analytics?.health_score ?? 0
    const rate = data.analytics?.today?.submission_rate
      ? Math.round(data.analytics.today.submission_rate * 100)
      : 0
    const subs = data.dashboard?.today_submissions ?? 0
    const total = data.dashboard?.employee_count ?? 0
    lines.push(`Health Score: ${hs}/100`)
    lines.push(`Today Submissions: ${subs}/${total} (${rate}%)`)

    if (data.analytics?.trend_7d?.length) {
      const trend = data.analytics.trend_7d
        .map((t) => `${Math.round(t.rate * 100)}%`)
        .join(', ')
      lines.push(`Trend 7d: [${trend}]`)
    }

    if (data.analytics?.sentiment) {
      const s = data.analytics.sentiment
      const parts = Object.entries(s).map(([k, v]) => `${k}: ${v}`)
      lines.push(`Sentiment: ${parts.join(', ')}`)
    }

    // Alerts
    lines.push('')
    lines.push(`=== ALERTS (${data.alerts.length} active) ===`)
    if (data.alerts.length === 0) {
      lines.push('No active alerts.')
    } else {
      for (const a of data.alerts) {
        const sev = a.severity === 'critical' ? 'CRITICAL' : 'WARNING'
        lines.push(`- [${sev}] ${a.employee_name}: ${a.message}`)
      }
    }

    // Employee Activity
    if (data.activity.length > 0) {
      lines.push('')
      lines.push('=== EMPLOYEE ACTIVITY ===')
      lines.push('| Name | Submitted 7d | Missed 7d | Sentiment |')
      lines.push('|------|-------------|----------|-----------|')
      for (const e of data.activity) {
        lines.push(`| ${e.name} | ${e.submitted_7d} | ${e.missed_7d} | ${e.last_sentiment} |`)
      }
    }

    // Goals
    if (data.goals.length > 0) {
      lines.push('')
      lines.push('=== GOALS (current cycle) ===')
      for (const g of data.goals) {
        const krs = g.key_results
        let progress = 0
        if (krs.length > 0) {
          const sum = krs.reduce((acc, kr) => {
            return acc + (kr.target > 0 ? Math.min((kr.current_value / kr.target) * 100, 100) : 0)
          }, 0)
          progress = Math.round(sum / krs.length)
        }
        const krDetail = krs.map((kr) => {
          const p = kr.target > 0 ? Math.round((kr.current_value / kr.target) * 100) : 0
          return `${kr.title}: ${p}%`
        }).join(', ')
        lines.push(`- "${g.title}" [${g.status}] ${progress}%${krDetail ? ` — ${krDetail}` : ''}`)
      }
    }

    // Board Records
    if (data.boardRecords.length > 0) {
      lines.push('')
      lines.push(`=== RECENT BOARD DISCUSSIONS (last ${data.boardRecords.length}) ===`)
      for (const r of data.boardRecords) {
        const date = new Date(r.created_at).toLocaleDateString()
        const preview = r.synthesis.slice(0, 100).replace(/\n/g, ' ')
        lines.push(`- "${r.topic}" (${date}): ${preview}...`)
      }
    }

    return lines.join('\n')
  }

  function summarize(data: CollectedData): string {
    const hs = data.analytics?.health_score ?? 0
    const alerts = data.alerts.length
    const rate = data.analytics?.today?.submission_rate
      ? Math.round(data.analytics.today.submission_rate * 100)
      : 0
    return `Health ${hs}, ${alerts} alerts, ${rate}% submission rate`
  }

  return { collect, formatContext, summarize }
}
