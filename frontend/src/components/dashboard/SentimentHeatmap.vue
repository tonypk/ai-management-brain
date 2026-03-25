<script setup lang="ts">
import { computed } from 'vue'
import { NCard } from 'naive-ui'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { HeatmapChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, VisualMapComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { EmployeeActivity } from '@/types'

use([HeatmapChart, GridComponent, TooltipComponent, VisualMapComponent, CanvasRenderer])

const props = defineProps<{
  employees: EmployeeActivity[]
}>()

const sentimentToValue: Record<string, number> = {
  positive: 3,
  neutral: 2,
  mixed: 1,
  negative: 0,
}

const option = computed(() => {
  const names = props.employees.map(e => e.name)
  const data = props.employees.map((e, idx) => [0, idx, sentimentToValue[e.last_sentiment] ?? 2])

  return {
    tooltip: {
      formatter: (p: { value: number[] }) => {
        const labels = ['Negative', 'Mixed', 'Neutral', 'Positive']
        return `${names[p.value[1]]}: ${labels[p.value[2]] || 'Unknown'}`
      },
    },
    grid: { left: 120, right: 40, bottom: 30, top: 10 },
    xAxis: {
      type: 'category' as const,
      data: ['Current'],
      splitArea: { show: true },
    },
    yAxis: {
      type: 'category' as const,
      data: names,
      splitArea: { show: true },
    },
    visualMap: {
      min: 0,
      max: 3,
      show: true,
      orient: 'horizontal' as const,
      left: 'center',
      bottom: 0,
      inRange: {
        color: ['#ef4444', '#f59e0b', '#94a3b8', '#22c55e'],
      },
      text: ['Positive', 'Negative'],
      textStyle: { fontSize: 11 },
    },
    series: [
      {
        type: 'heatmap' as const,
        data,
        label: { show: false },
        emphasis: { itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0,0,0,0.5)' } },
      },
    ],
  }
})
</script>

<template>
  <NCard title="Sentiment Heatmap" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <div v-if="employees.length === 0" style="text-align: center; color: #888; padding: 40px">No employee data</div>
    <VChart v-else :option="option" autoresize :style="{ height: Math.max(200, employees.length * 36 + 60) + 'px' }" />
  </NCard>
</template>
