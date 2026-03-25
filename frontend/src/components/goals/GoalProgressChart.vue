<script setup lang="ts">
import { computed } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { Objective } from '@/types'

use([BarChart, GridComponent, TooltipComponent, CanvasRenderer])

const props = defineProps<{
  objectives: Objective[]
}>()

function objectiveProgress(obj: Objective): number {
  if (obj.key_results.length === 0) return 0
  const sum = obj.key_results.reduce((acc, kr) => {
    return acc + (kr.target > 0 ? Math.min((kr.current_value / kr.target) * 100, 100) : 0)
  }, 0)
  return Math.round(sum / obj.key_results.length)
}

function barColor(val: number): string {
  if (val >= 80) return '#22c55e'
  if (val >= 50) return '#f59e0b'
  return '#ef4444'
}

const option = computed(() => {
  const sorted = [...props.objectives].reverse()
  const names = sorted.map((o) => o.title.length > 25 ? o.title.slice(0, 22) + '...' : o.title)
  const values = sorted.map(objectiveProgress)

  return {
    tooltip: { trigger: 'axis' as const, formatter: '{b}: {c}%' },
    grid: { left: 120, right: 40, top: 10, bottom: 20 },
    xAxis: { type: 'value' as const, max: 100, axisLabel: { formatter: '{value}%' } },
    yAxis: { type: 'category' as const, data: names },
    series: [{
      type: 'bar' as const,
      data: values.map((v) => ({ value: v, itemStyle: { color: barColor(v) } })),
      barWidth: 16,
      label: { show: true, position: 'right' as const, formatter: '{c}%', fontSize: 12 },
    }],
  }
})
</script>

<template>
  <VChart
    v-if="objectives.length > 0"
    :option="option"
    autoresize
    :style="{ height: Math.max(objectives.length * 40 + 40, 120) + 'px', width: '100%' }"
  />
  <div v-else style="color: #999; padding: 20px; text-align: center; font-size: 13px">
    No objectives to chart
  </div>
</template>
