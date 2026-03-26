<script setup lang="ts">
import { NGrid, NGi } from 'naive-ui'
import {
  BusinessOutline as IndustryIcon,
  PeopleOutline as SizeIcon,
  TrendingUpOutline as StageIcon,
  CheckmarkCircleOutline as StatusIcon,
} from '@vicons/ionicons5'
import StatCard from '@/components/shared/StatCard.vue'
import type { OrgPlan } from '@/types'

defineProps<{ plan: OrgPlan }>()
</script>

<template>
  <NGrid :cols="4" :x-gap="16" :y-gap="16" responsive="screen" :item-responsive="true">
    <NGi :span="4" style="min-width: 0" >
      <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 16px">
        <StatCard label="Industry" :value="plan.industry" :icon="IndustryIcon" icon-color="#6366f1" />
        <StatCard label="Team Size" :value="plan.size" :icon="SizeIcon" icon-color="#0ea5e9" />
        <StatCard label="Stage" :value="plan.stage" :icon="StageIcon" icon-color="#f59e0b" />
        <StatCard
          label="Status"
          :value="plan.status === 'active' ? '● Active' : plan.status"
          :icon="StatusIcon"
          :icon-color="plan.status === 'active' ? '#22c55e' : '#94a3b8'"
        />
      </div>
    </NGi>
  </NGrid>
</template>
