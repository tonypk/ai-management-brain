<script setup lang="ts">
import { computed } from 'vue'
import { NSelect } from 'naive-ui'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [val: string]
}>()

function getCurrentQuarter(): { year: number; quarter: number } {
  const now = new Date()
  return { year: now.getFullYear(), quarter: Math.ceil((now.getMonth() + 1) / 3) }
}

const options = computed(() => {
  const { year, quarter } = getCurrentQuarter()
  const cycles: string[] = []

  // 3 past quarters
  for (let i = 3; i >= 1; i--) {
    let q = quarter - i
    let y = year
    while (q <= 0) { q += 4; y-- }
    cycles.push(`${y}-Q${q}`)
  }
  // current
  cycles.push(`${year}-Q${quarter}`)
  // 1 future
  let nq = quarter + 1
  let ny = year
  if (nq > 4) { nq = 1; ny++ }
  cycles.push(`${ny}-Q${nq}`)

  return cycles.map((c) => ({ label: c, value: c }))
})
</script>

<template>
  <NSelect
    :value="modelValue"
    :options="options"
    style="width: 160px"
    @update:value="emit('update:modelValue', $event)"
  />
</template>
