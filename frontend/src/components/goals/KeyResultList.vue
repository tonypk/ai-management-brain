<script setup lang="ts">
import { NProgress, NSpace, NText, NButton, NIcon } from 'naive-ui'
import { CreateOutline, TrashOutline } from '@vicons/ionicons5'
import type { KeyResult } from '@/types'

defineProps<{
  keyResults: KeyResult[]
  objectiveId: string
}>()

const emit = defineEmits<{
  edit: [kr: KeyResult]
  delete: [krId: string]
  updateProgress: [krId: string, value: number]
}>()

function krPercent(kr: KeyResult): number {
  if (kr.target <= 0) return 0
  return Math.min(Math.round((kr.current_value / kr.target) * 100), 100)
}

function progressStatus(pct: number): 'success' | 'warning' | 'error' | 'default' {
  if (pct >= 100) return 'success'
  if (pct >= 50) return 'warning'
  if (pct > 0) return 'error'
  return 'default'
}
</script>

<template>
  <div>
    <div
      v-for="kr in keyResults"
      :key="kr.id"
      style="padding: 8px 0; border-bottom: 1px solid #f5f5f5"
    >
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px">
        <NText style="font-size: 13px">{{ kr.title }}</NText>
        <NSpace :size="2">
          <NButton size="tiny" text @click="emit('edit', kr)">
            <template #icon><NIcon :component="CreateOutline" :size="14" /></template>
          </NButton>
          <NButton size="tiny" text type="error" @click="emit('delete', kr.id)">
            <template #icon><NIcon :component="TrashOutline" :size="14" /></template>
          </NButton>
        </NSpace>
      </div>
      <div style="display: flex; align-items: center; gap: 8px">
        <NProgress
          type="line"
          :percentage="krPercent(kr)"
          :status="progressStatus(krPercent(kr))"
          :show-indicator="false"
          style="flex: 1"
        />
        <NText depth="3" style="font-size: 12px; white-space: nowrap">
          {{ kr.current_value }}/{{ kr.target }} {{ kr.unit }}
        </NText>
      </div>
    </div>
  </div>
</template>
