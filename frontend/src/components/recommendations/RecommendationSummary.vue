<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NCard, NButton, NTag, NSpace, NText, NEmpty } from 'naive-ui'
import { useRouter } from 'vue-router'
import { getRecommendationSummary, executeAction, dismissRecommendation } from '@/api/recommendations'
import type { RecommendationSummary } from '@/types'

const router = useRouter()
const summary = ref<RecommendationSummary | null>(null)

const priorityColor: Record<string, string> = {
  critical: 'error',
  high: 'warning',
  medium: 'info',
  low: 'default',
}

onMounted(async () => {
  try {
    summary.value = await getRecommendationSummary()
  } catch { /* ignore */ }
})

async function handleQuickExecute(recId: string) {
  try {
    await executeAction(recId, 0)
    summary.value = await getRecommendationSummary()
  } catch { /* ignore */ }
}

async function handleQuickDismiss(recId: string) {
  try {
    await dismissRecommendation(recId)
    summary.value = await getRecommendationSummary()
  } catch { /* ignore */ }
}
</script>

<template>
  <NCard v-if="summary && summary.pending_count > 0" :bordered="false" size="small">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px">
      <NText strong style="font-size: 14px">AI Recommendations</NText>
      <NButton text type="primary" size="small" @click="router.push('/recommendations')">
        View All ({{ summary.pending_count }}) ->
      </NButton>
    </div>

    <div v-for="rec in summary.top" :key="rec.id" style="padding: 8px 0; border-top: 1px solid #f0f0f0">
      <NSpace :size="6" align="center" style="margin-bottom: 4px">
        <NTag :type="(priorityColor[rec.priority] as any)" size="tiny">{{ rec.priority }}</NTag>
        <NText style="font-size: 13px; font-weight: 500">{{ rec.title }}</NText>
      </NSpace>
      <div style="display: flex; gap: 6px; margin-top: 4px">
        <NButton size="tiny" type="primary" secondary @click="handleQuickExecute(rec.id)">
          {{ rec.suggested_actions[0]?.label || 'Execute' }}
        </NButton>
        <NButton size="tiny" quaternary @click="handleQuickDismiss(rec.id)">Dismiss</NButton>
      </div>
    </div>

    <NEmpty v-if="summary.top.length === 0" description="No recommendations" />
  </NCard>
</template>
