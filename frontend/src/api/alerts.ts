import { get } from './client'
import type { Alert, CheckinStatus } from '@/types'

export async function getAlerts(): Promise<Alert[]> {
  const res = await get<{ data: Alert[] }>('/openclaw/alerts')
  return res.data
}

export async function getCheckinStatus(): Promise<CheckinStatus> {
  const res = await get<{ data: CheckinStatus }>('/openclaw/status')
  return res.data
}
