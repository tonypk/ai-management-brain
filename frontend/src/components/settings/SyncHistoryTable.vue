<script setup lang="ts">
import { h } from 'vue'
import { NCard, NDataTable, NTag, type DataTableColumn } from 'naive-ui'
import type { SyncLog } from '@/types/sync'

defineProps<{
  logs: SyncLog[]
  loading: boolean
}>()

function formatTime(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatDuration(started: string, completed: string | null): string {
  if (!completed) return '-'
  const ms = new Date(completed).getTime() - new Date(started).getTime()
  if (ms < 1000) return `${ms}ms`
  const secs = Math.floor(ms / 1000)
  if (secs < 60) return `${secs}s`
  const mins = Math.floor(secs / 60)
  const remSecs = secs % 60
  return `${mins}m ${remSecs}s`
}

function statusType(status: string): 'success' | 'warning' | 'error' | 'info' {
  if (status === 'success') return 'success'
  if (status === 'partial') return 'warning'
  if (status === 'failed') return 'error'
  return 'info'
}

const columns: DataTableColumn<SyncLog>[] = [
  {
    title: 'Time',
    key: 'started_at',
    width: 160,
    render(row) {
      return formatTime(row.started_at)
    },
  },
  {
    title: 'Status',
    key: 'status',
    width: 100,
    render(row) {
      return h(NTag, { type: statusType(row.status), size: 'small' }, () => row.status)
    },
  },
  {
    title: 'Pushed',
    key: 'items_pushed',
    width: 80,
    align: 'center',
  },
  {
    title: 'Pulled',
    key: 'items_pulled',
    width: 80,
    align: 'center',
  },
  {
    title: 'Conflicts',
    key: 'conflicts',
    width: 90,
    align: 'center',
    render(row) {
      if (row.conflicts > 0) {
        return h(NTag, { type: 'warning', size: 'small' }, () => String(row.conflicts))
      }
      return '0'
    },
  },
  {
    title: 'Duration',
    key: 'duration',
    width: 100,
    render(row) {
      return formatDuration(row.started_at, row.completed_at)
    },
  },
]
</script>

<template>
  <NCard title="Sync History" size="small" :bordered="true">
    <NDataTable
      :columns="columns"
      :data="logs"
      :loading="loading"
      :bordered="false"
      size="small"
      :max-height="320"
      :pagination="{ pageSize: 10 }"
    />
  </NCard>
</template>
