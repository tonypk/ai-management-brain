<script setup lang="ts">
import { computed } from 'vue'
import { NGrid, NGi } from 'naive-ui'
import {
  CheckmarkCircleOutline,
  CloseCircleOutline,
  HappyOutline,
  SpeedometerOutline,
} from '@vicons/ionicons5'
import StatCard from '@/components/shared/StatCard.vue'
import type { EmployeeActivity } from '@/types'

const props = defineProps<{
  activity: EmployeeActivity | null
}>()

const submissionRate = computed(() => {
  if (!props.activity) return '0%'
  const total = props.activity.submitted_7d + props.activity.missed_7d
  if (total === 0) return '0%'
  return Math.round((props.activity.submitted_7d / total) * 100) + '%'
})
</script>

<template>
  <NGrid :cols="24" :x-gap="12" :y-gap="12" responsive="screen" :item-responsive="true">
    <NGi span="12 m:6">
      <StatCard
        label="Submitted (7d)"
        :value="activity?.submitted_7d ?? 0"
        :icon="CheckmarkCircleOutline"
        icon-color="#22c55e"
      />
    </NGi>
    <NGi span="12 m:6">
      <StatCard
        label="Missed (7d)"
        :value="activity?.missed_7d ?? 0"
        :icon="CloseCircleOutline"
        icon-color="#ef4444"
      />
    </NGi>
    <NGi span="12 m:6">
      <StatCard
        label="Sentiment"
        :value="activity?.last_sentiment ?? '-'"
        :icon="HappyOutline"
        icon-color="#f59e0b"
      />
    </NGi>
    <NGi span="12 m:6">
      <StatCard
        label="Rate (7d)"
        :value="submissionRate"
        :icon="SpeedometerOutline"
        icon-color="#6366f1"
      />
    </NGi>
  </NGrid>
</template>
