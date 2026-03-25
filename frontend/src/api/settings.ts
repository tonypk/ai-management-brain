import { get, put, post } from './client'
import type { Tenant, ChannelConfig, SchedulerJob, BillingStatus } from '@/types'

// Tenant
export async function getTenant(): Promise<Tenant> {
  const res = await get<{ data: Tenant }>('/tenant')
  return res.data
}

export async function updateTenant(name: string, timezone: string): Promise<void> {
  await put<{ data: unknown }>('/tenant', { name, timezone })
}

// Channels
export async function getChannelConfig(): Promise<ChannelConfig> {
  const res = await get<{ data: ChannelConfig }>('/admin/channels')
  return res.data
}

export async function updateChannelConfig(data: {
  enabled_channels?: string[]
  slack_bot_token?: string
  slack_signing_secret?: string
  lark_app_id?: string
  lark_app_secret?: string
  signal_phone?: string
}): Promise<void> {
  await put<{ data: { updated: boolean } }>('/admin/channels', data)
}

export async function testChannel(channel: string, userId: string, text?: string): Promise<boolean> {
  const res = await post<{ data: { sent: boolean } }>(`/admin/channels/test/${channel}`, { user_id: userId, text })
  return res.data.sent
}

// Scheduler
export async function listSchedulerJobs(): Promise<SchedulerJob[]> {
  const res = await get<{ data: SchedulerJob[] }>('/admin/scheduler')
  return res.data
}

export async function updateJobSchedule(job: string, cron: string): Promise<void> {
  await put<{ data: { job: string; cron: string } }>(`/admin/scheduler/${job}/schedule`, { cron })
}

export async function triggerJob(job: string): Promise<string> {
  const res = await post<{ data: { triggered: string } }>(`/admin/scheduler/${job}/trigger`)
  return res.data.triggered
}

// Billing
export async function getBillingStatus(): Promise<BillingStatus> {
  const res = await get<{ data: BillingStatus }>('/billing/status')
  return res.data
}
