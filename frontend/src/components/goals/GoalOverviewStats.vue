<script setup lang="ts">
import { NGrid, NGi } from 'naive-ui'
import {
  FlagOutline as TotalIcon,
  FlashOutline as ActiveIcon,
  CheckmarkCircleOutline as CompletedIcon,
} from '@vicons/ionicons5'
import HealthGauge from '@/components/shared/HealthGauge.vue'
import StatCard from '@/components/shared/StatCard.vue'

defineProps<{
  progress: number
  total: number
  active: number
  completed: number
}>()
</script>

<template>
  <NGrid :x-gap="16" :y-gap="16" cols="2 m:4" responsive="screen">
    <NGi>
      <div style="display: flex; justify-content: center">
        <HealthGauge :value="progress" title="Progress" />
      </div>
    </NGi>
    <NGi>
      <StatCard label="Total Objectives" :value="total" :icon="TotalIcon" icon-color="#6366f1" />
    </NGi>
    <NGi>
      <StatCard label="Active" :value="active" :icon="ActiveIcon" icon-color="#3b82f6" />
    </NGi>
    <NGi>
      <StatCard label="Completed" :value="completed" :icon="CompletedIcon" icon-color="#22c55e" />
    </NGi>
  </NGrid>
</template>
