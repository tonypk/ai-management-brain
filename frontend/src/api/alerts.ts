import { get } from './client'
import type { Alert, CheckinStatus } from '@/types'

export async function getAlerts(): Promise<Alert[]> {
  const res = await get<{ alerts: Alert[]; total: number }>('/openclaw/alerts')
  return res.alerts ?? []
}

export async function getCheckinStatus(): Promise<CheckinStatus> {
  return get<CheckinStatus>('/openclaw/status')
}
