<script setup lang="ts">
import { computed } from 'vue'
import { NSelect, NSpace } from 'naive-ui'
import type { DigestPeriod } from '@/types'

const props = defineProps<{
  period: DigestPeriod
  selectedLabel: string
}>()

const emit = defineEmits<{
  'update:period': [val: DigestPeriod]
  'update:selectedLabel': [val: string]
}>()

const periodOptions = [
  { label: 'Weekly', value: 'weekly' as DigestPeriod },
  { label: 'Monthly', value: 'monthly' as DigestPeriod },
]

function getWeekOptions(): { label: string; value: string }[] {
  const opts: { label: string; value: string }[] = []
  const now = new Date()
  for (let i = 0; i < 8; i++) {
    const d = new Date(now)
    d.setDate(d.getDate() - i * 7)
    const monday = new Date(d)
    monday.setDate(monday.getDate() - ((monday.getDay() + 6) % 7))
    const label = `Week of ${monday.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}`
    if (!opts.find((o) => o.value === label)) {
      opts.push({ label, value: label })
    }
  }
  return opts
}

function getMonthOptions(): { label: string; value: string }[] {
  const opts: { label: string; value: string }[] = []
  const now = new Date()
  for (let i = 0; i < 6; i++) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
    const label = d.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })
    opts.push({ label, value: label })
  }
  return opts
}

const labelOptions = computed(() =>
  props.period === 'weekly' ? getWeekOptions() : getMonthOptions(),
)

function handlePeriodChange(val: DigestPeriod) {
  emit('update:period', val)
  const opts = val === 'weekly' ? getWeekOptions() : getMonthOptions()
  if (opts.length > 0) {
    emit('update:selectedLabel', opts[0].value)
  }
}
</script>

<template>
  <NSpace :size="12" align="center">
    <NSelect
      :value="period"
      :options="periodOptions"
      style="width: 120px"
      @update:value="handlePeriodChange"
    />
    <NSelect
      :value="selectedLabel"
      :options="labelOptions"
      style="width: 260px"
      @update:value="emit('update:selectedLabel', $event)"
    />
  </NSpace>
</template>
