<script setup lang="ts">
import { NCard, NList, NListItem, NSpace, NButton } from 'naive-ui'
import SeverityBadge from '@/components/shared/SeverityBadge.vue'
import type { Alert } from '@/types'
import { useRouter } from 'vue-router'

const router = useRouter()

defineProps<{
  alerts: Alert[]
}>()
</script>

<template>
  <NCard
    title="Active Alerts"
    :bordered="false"
    style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)"
  >
    <template #header-extra>
      <NButton text type="primary" size="small" @click="router.push('/alerts')">
        View All
      </NButton>
    </template>
    <NList v-if="alerts.length > 0" :bordered="false">
      <NListItem v-for="(alert, idx) in alerts.slice(0, 5)" :key="idx">
        <NSpace align="center" :size="8">
          <SeverityBadge :severity="alert.severity" />
          <span style="font-weight: 500">{{ alert.employee_name }}</span>
          <span style="color: #888; font-size: 13px">{{ alert.message }}</span>
        </NSpace>
      </NListItem>
    </NList>
    <div v-else style="text-align: center; color: #888; padding: 32px">
      No active alerts
    </div>
  </NCard>
</template>
