export interface ChannelStatus {
  type: string
  configured: boolean
}

export interface ChannelConfig {
  enabled_channels: string[]
  channels: ChannelStatus[]
  registered_channels: string[]
}

export interface MemoryItem {
  id: string
  tenant_id: string
  memory_type: string
  memory_tier: string
  employee_id: string | null
  content: string
  summary: string | null
  importance: number
  access_count: number
  metadata: Record<string, unknown>
  expires_at: string | null
  created_at: string
  updated_at: string
}

export interface MemoryStats {
  total: number
}

export interface GroupChat {
  id: string
  platform: string
  platform_chat_id: string
  name: string
  group_type: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface SchedulerJob {
  name: string
  cron: string
  last_run: string
  next_run: string
}
