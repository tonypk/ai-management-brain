<script setup lang="ts">
import { NCard, NTag, NSpace, NEmpty } from 'naive-ui'
import type { InsightRow } from '@/api/worldmodel'

defineProps<{ insights: InsightRow[] }>()

const dimensionColors: Record<string, string> = {
  risk: 'error',
  opportunity: 'success',
  rhythm: 'info',
  context: 'warning',
}
</script>

<template>
  <NEmpty v-if="!insights.length" description="No insights yet. Check-in data will generate insights." />
  <NSpace v-else vertical>
    <NCard v-for="(insight, i) in insights" :key="i" size="small">
      <template #header>
        <NTag :type="dimensionColors[insight.dimension] || 'default'" size="small">
          {{ insight.dimension }}
        </NTag>
      </template>
      <p>{{ insight.insight_text }}</p>
      <template #header-extra>
        <span style="font-size: 12px; color: #999">{{ Math.round(insight.confidence * 100) }}%</span>
      </template>
    </NCard>
  </NSpace>
</template>
