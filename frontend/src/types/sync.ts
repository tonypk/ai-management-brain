export interface SyncConfig {
  id: string
  tenant_id: string
  storage_type: 'notion' | 'sheets'
  is_enabled: boolean
  entity_types: string[]
  sync_frequency_minutes: number
  last_sync_at: string | null
  last_sync_status: string | null
  config: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface SyncLog {
  id: string
  tenant_id: string
  sync_config_id: string
  direction: string
  started_at: string
  completed_at: string | null
  status: string
  items_pushed: number
  items_pulled: number
  conflicts: number
  errors: unknown[]
  summary: string | null
}

export interface ConfigureSyncRequest {
  storage_type: 'notion' | 'sheets'
  is_enabled: boolean
  entity_types: string[]
  sync_frequency_minutes?: number
  config?: Record<string, unknown>
}
