<script setup lang="ts">
import { NCard, NButton, NIcon, NSpace, NText, useMessage } from 'naive-ui'
import { CopyOutline, RefreshOutline } from '@vicons/ionicons5'
import type { InsightRecord } from '@/types'

defineProps<{
  insight: InsightRecord | null
  loading: boolean
}>()

const emit = defineEmits<{
  regenerate: []
}>()

const message = useMessage()

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString()
}

async function handleCopy(content: string) {
  try {
    await navigator.clipboard.writeText(content)
    message.success('Copied to clipboard')
  } catch {
    message.error('Failed to copy')
  }
}
</script>

<template>
  <NCard
    :bordered="false"
    style="box-shadow: 0 1px 3px rgba(0,0,0,0.08); margin-top: 16px"
  >
    <template #header>
      <NSpace justify="space-between" align="center" style="width: 100%">
        <span style="font-weight: 600">Latest Insight</span>
        <NText v-if="insight" depth="3" style="font-size: 13px">
          {{ formatDate(insight.created_at) }}
        </NText>
      </NSpace>
    </template>

    <div v-if="loading" style="text-align: center; padding: 40px 0; color: #888">
      Analyzing team data with AI...
    </div>

    <div v-else-if="insight">
      <div style="white-space: pre-wrap; line-height: 1.7; font-size: 14px">{{ insight.content }}</div>
      <NSpace justify="end" style="margin-top: 16px">
        <NButton size="small" quaternary @click="handleCopy(insight.content)">
          <template #icon><NIcon :component="CopyOutline" /></template>
          Copy
        </NButton>
        <NButton size="small" quaternary @click="emit('regenerate')">
          <template #icon><NIcon :component="RefreshOutline" /></template>
          Regenerate
        </NButton>
      </NSpace>
    </div>

    <div v-else style="text-align: center; padding: 40px 0; color: #888">
      Click "Generate Insights" to analyze your team data with AI.
    </div>
  </NCard>
</template>
