<script setup lang="ts">
import { NCard, NButton, NTag, NSpace, NText, NPopconfirm, useMessage } from 'naive-ui'
import type { Recommendation } from '@/types'
import { executeAction, executeAll, dismissRecommendation } from '@/api/recommendations'

const props = defineProps<{
  recommendation: Recommendation
}>()

const emit = defineEmits<{
  refresh: []
}>()

const message = useMessage()

const priorityColor: Record<string, string> = {
  critical: 'error',
  high: 'warning',
  medium: 'info',
  low: 'default',
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

async function handleExecute(index: number) {
  try {
    const result = await executeAction(props.recommendation.id, index)
    if (result.success) {
      message.success(result.message || 'Action executed')
    } else if (result.needs_confirmation) {
      message.warning('This action requires confirmation on the web')
    } else if (result.link) {
      window.location.hash = result.link
    } else {
      message.error(result.error || 'Execution failed')
    }
    emit('refresh')
  } catch {
    message.error('Failed to execute action')
  }
}

async function handleExecuteAll() {
  try {
    const result = await executeAll(props.recommendation.id)
    if (result.all_done) {
      message.success('All actions executed')
    } else {
      const succeeded = result.results.filter(r => r.success).length
      message.info(`${succeeded}/${result.results.length} actions executed`)
    }
    emit('refresh')
  } catch {
    message.error('Failed to execute actions')
  }
}

async function handleDismiss() {
  try {
    await dismissRecommendation(props.recommendation.id)
    message.info('Recommendation dismissed')
    emit('refresh')
  } catch {
    message.error('Failed to dismiss')
  }
}
</script>

<template>
  <NCard :bordered="false" size="small" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08); margin-bottom: 12px">
    <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 8px">
      <div style="flex: 1; min-width: 0">
        <NSpace :size="8" align="center" style="margin-bottom: 4px">
          <NTag :type="(priorityColor[recommendation.priority] as any)" size="small">
            {{ recommendation.priority }}
          </NTag>
          <NTag size="small">{{ recommendation.category }}</NTag>
        </NSpace>
        <NText strong style="font-size: 14px">{{ recommendation.title }}</NText>
      </div>
      <NText depth="3" style="font-size: 11px; white-space: nowrap; margin-left: 8px">
        {{ timeAgo(recommendation.created_at) }}
      </NText>
    </div>

    <NText depth="2" style="font-size: 13px; display: block; margin-bottom: 8px">
      {{ recommendation.description }}
    </NText>

    <!-- Evidence tags -->
    <NSpace :size="4" style="margin-bottom: 8px" v-if="recommendation.evidence">
      <NTag v-for="s in (recommendation.evidence.signals || [])" :key="s.name" size="tiny" round>
        {{ s.name }}: {{ s.value }}
      </NTag>
      <NTag v-for="e in (recommendation.evidence.employees || [])" :key="e.name" size="tiny" round type="warning">
        {{ e.name }}: {{ e.issue }}
      </NTag>
      <NTag v-for="m in (recommendation.evidence.metrics || [])" :key="m.name" size="tiny" round type="info">
        {{ m.name }}: {{ m.trend }}
      </NTag>
    </NSpace>

    <!-- Actions -->
    <div v-if="recommendation.status === 'pending'" style="display: flex; gap: 8px; flex-wrap: wrap">
      <NButton
        v-for="(action, i) in recommendation.suggested_actions"
        :key="i"
        size="small"
        type="primary"
        secondary
        @click="handleExecute(i)"
      >
        {{ action.label }}
      </NButton>
      <div style="flex: 1" />
      <NButton v-if="recommendation.suggested_actions.length > 1" size="small" type="primary" @click="handleExecuteAll">
        Execute All
      </NButton>
      <NPopconfirm @positive-click="handleDismiss">
        <template #trigger>
          <NButton size="small" quaternary>Dismiss</NButton>
        </template>
        Dismiss this recommendation?
      </NPopconfirm>
    </div>

    <!-- Executed/Dismissed status -->
    <NTag v-else-if="recommendation.status === 'executed'" type="success" size="small">
      Executed {{ recommendation.executed_at ? timeAgo(recommendation.executed_at) : '' }}
    </NTag>
    <NTag v-else-if="recommendation.status === 'dismissed'" size="small">
      Dismissed
    </NTag>

    <div style="font-size: 11px; color: #999; margin-top: 6px">
      Source: {{ recommendation.source === 'daily_scan' ? 'Daily Analysis' : 'Real-time Trigger' }}
    </div>
  </NCard>
</template>
