<script setup lang="ts">
import { type Component } from 'vue'
import { NCard, NStatistic, NIcon } from 'naive-ui'

defineProps<{
  label: string
  value: string | number
  icon?: Component
  iconColor?: string
}>()
</script>

<template>
  <NCard size="small" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <div style="display: flex; align-items: center; gap: 16px">
      <div
        v-if="icon"
        style="width: 48px; height: 48px; border-radius: 12px; display: flex; align-items: center; justify-content: center"
        :style="{ background: (iconColor || '#6366f1') + '15', color: iconColor || '#6366f1' }"
      >
        <NIcon :size="24" :component="icon" />
      </div>
      <NStatistic :label="label" :value="value" />
    </div>
  </NCard>
</template>
