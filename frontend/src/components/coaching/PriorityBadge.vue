<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'
import type { RiskLevel } from '@/types'

const props = defineProps<{
  level: RiskLevel
}>()

const config = computed(() => {
  const map: Record<RiskLevel, { type: 'error' | 'warning' | 'info' | 'default'; label: string }> = {
    high: { type: 'error', label: 'HIGH' },
    medium: { type: 'warning', label: 'MED' },
    low: { type: 'info', label: 'LOW' },
    none: { type: 'default', label: '-' },
  }
  return map[props.level] || map.none
})
</script>

<template>
  <NTag :type="config.type" size="small" round>
    {{ config.label }}
  </NTag>
</template>
