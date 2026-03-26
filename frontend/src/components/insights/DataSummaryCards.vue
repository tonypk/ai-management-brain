<script setup lang="ts">
import { NGrid, NGi } from 'naive-ui'
import {
  SpeedometerOutline,
  AlertCircleOutline,
  CheckmarkCircleOutline,
  WarningOutline,
} from '@vicons/ionicons5'
import StatCard from '@/components/shared/StatCard.vue'

defineProps<{
  healthScore: number
  alertCount: number
  submissionRate: number
  atRiskCount: number
}>()
</script>

<template>
  <NGrid :cols="24" :x-gap="16" :y-gap="16" responsive="screen" :item-responsive="true">
    <NGi span="24 m:6">
      <StatCard
        label="Health Score"
        :value="healthScore"
        :icon="SpeedometerOutline"
        icon-color="#6366f1"
      />
    </NGi>
    <NGi span="24 m:6">
      <StatCard
        label="Active Alerts"
        :value="alertCount"
        :icon="AlertCircleOutline"
        :icon-color="alertCount > 0 ? '#ef4444' : '#22c55e'"
      />
    </NGi>
    <NGi span="24 m:6">
      <StatCard
        label="Submission Rate"
        :value="submissionRate + '%'"
        :icon="CheckmarkCircleOutline"
        icon-color="#22c55e"
      />
    </NGi>
    <NGi span="24 m:6">
      <StatCard
        label="At Risk"
        :value="atRiskCount"
        :icon="WarningOutline"
        :icon-color="atRiskCount > 0 ? '#f59e0b' : '#22c55e'"
      />
    </NGi>
  </NGrid>
</template>
