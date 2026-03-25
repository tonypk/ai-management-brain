import { get } from './client'
import type { Report, Summary } from '@/types'

export async function listReports(date: string): Promise<Report[]> {
  const res = await get<{ data: Report[] }>(`/reports?date=${date}`)
  return res.data
}

export async function getSummary(date: string): Promise<Summary> {
  const res = await get<{ data: Summary }>(`/reports/summary?date=${date}`)
  return res.data
}
