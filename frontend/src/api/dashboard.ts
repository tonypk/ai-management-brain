import { get } from './client'
import type { DashboardStats, AnalyticsOverview, EmployeeActivity } from '@/types'

export async function getDashboard(): Promise<DashboardStats> {
  const res = await get<{ data: DashboardStats }>('/dashboard')
  return res.data
}

export async function getAnalyticsOverview(): Promise<AnalyticsOverview> {
  const res = await get<{ data: AnalyticsOverview }>('/analytics/overview')
  return res.data
}

export async function getEmployeeActivity(): Promise<EmployeeActivity[]> {
  const res = await get<{ data: EmployeeActivity[] }>('/analytics/activity')
  return res.data
}
