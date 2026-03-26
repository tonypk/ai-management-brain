import { get, put, post } from './client'
import type { SyncConfig, SyncLog, ConfigureSyncRequest } from '@/types/sync'

export async function listSyncConfigs(): Promise<SyncConfig[]> {
  const res = await get<{ data: SyncConfig[] }>('/sync/configs')
  return res.data
}

export async function configureSync(data: ConfigureSyncRequest): Promise<SyncConfig> {
  const res = await put<{ data: SyncConfig }>('/sync/config', data)
  return res.data
}

export async function triggerSync(configId: string): Promise<void> {
  await post<{ data: unknown }>(`/sync/${configId}/trigger`)
}

export async function listSyncLogs(configId: string, limit = 20): Promise<SyncLog[]> {
  const res = await get<{ data: SyncLog[] }>(`/sync/logs?config_id=${encodeURIComponent(configId)}&limit=${limit}`)
  return res.data
}
