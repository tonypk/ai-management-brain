<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'
import type { GoalStatus } from '@/types'

const props = defineProps<{
  status: GoalStatus
}>()

const config = computed(() => {
  const map: Record<GoalStatus, { label: string; type: 'default' | 'info' | 'success' | 'warning' | 'error' }> = {
    draft: { label: 'Draft', type: 'default' },
    active: { label: 'Active', type: 'info' },
    completed: { label: 'Completed', type: 'success' },
    cancelled: { label: 'Cancelled', type: 'warning' },
  }
  return map[props.status]
})
</script>

<template>
  <NTag :type="config.type" size="small" :bordered="false">{{ config.label }}</NTag>
</template>
