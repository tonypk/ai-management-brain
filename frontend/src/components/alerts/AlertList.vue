<script setup lang="ts">
import { NList, NListItem, NThing, NSpace, NTag } from 'naive-ui'
import SeverityBadge from '@/components/shared/SeverityBadge.vue'
import type { Alert } from '@/types'

defineProps<{
  alerts: Alert[]
}>()
</script>

<template>
  <NList v-if="alerts.length > 0" bordered>
    <NListItem v-for="(alert, idx) in alerts" :key="idx">
      <NThing>
        <template #header>
          <NSpace align="center" :size="8">
            <SeverityBadge :severity="alert.severity" />
            <span style="font-weight: 600">{{ alert.employee_name }}</span>
          </NSpace>
        </template>
        <template #description>
          <div style="margin-top: 4px; color: #555">{{ alert.message }}</div>
          <NSpace :size="12" style="margin-top: 8px">
            <NTag v-if="alert.consecutive_misses" size="small" type="error">
              {{ alert.consecutive_misses }} consecutive misses
            </NTag>
            <NTag v-if="alert.chase_count" size="small" type="warning">
              Chased {{ alert.chase_count }}x
            </NTag>
            <NTag v-if="alert.last_checkin" size="small">
              Last: {{ alert.last_checkin }}
            </NTag>
          </NSpace>
        </template>
      </NThing>
    </NListItem>
  </NList>
  <div v-else style="text-align: center; color: #888; padding: 48px">
    No active alerts — all clear!
  </div>
</template>
