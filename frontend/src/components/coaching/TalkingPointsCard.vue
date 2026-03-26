<script setup lang="ts">
import { NCard, NEmpty } from 'naive-ui'
import PriorityBadge from './PriorityBadge.vue'
import type { TalkingPoint } from '@/types'

defineProps<{
  points: TalkingPoint[]
}>()
</script>

<template>
  <NCard title="Talking Points" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <template v-if="points.length">
      <div v-for="tp in points" :key="tp.employee_name" style="margin-bottom: 16px">
        <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px">
          <span style="font-weight: 600; font-size: 14px">{{ tp.employee_name }}</span>
          <PriorityBadge :level="tp.priority" />
        </div>
        <ul style="margin: 0; padding-left: 20px">
          <li v-for="(point, idx) in tp.points" :key="idx" style="font-size: 13px; color: #555; margin-bottom: 4px">
            {{ point }}
          </li>
        </ul>
      </div>
    </template>
    <NEmpty v-else description="No talking points — all employees are on track" />
  </NCard>
</template>
