<script setup lang="ts">
import { computed } from 'vue'
import { NCard } from 'naive-ui'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { BarChart, LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { TrendPoint } from '@/types'

use([BarChart, LineChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

const props = defineProps<{
  data: TrendPoint[]
}>()

const option = computed(() => {
  const dates = props.data.map(d => d.date.slice(5))
  const counts = props.data.map(d => d.count)
  const rates = props.data.map(d => d.rate)

  return {
    tooltip: { trigger: 'axis' as const },
    legend: { data: ['Submissions', 'Rate %'], top: 0 },
    grid: { left: 50, right: 50, bottom: 30, top: 40 },
    xAxis: { type: 'category' as const, data: dates },
    yAxis: [
      { type: 'value' as const, name: 'Count', min: 0 },
      { type: 'value' as const, name: 'Rate %', min: 0, max: 100 },
    ],
    series: [
      {
        name: 'Submissions',
        type: 'bar' as const,
        data: counts,
        itemStyle: { color: '#6366f1', borderRadius: [4, 4, 0, 0] },
        barMaxWidth: 32,
      },
      {
        name: 'Rate %',
        type: 'line' as const,
        yAxisIndex: 1,
        data: rates,
        smooth: true,
        lineStyle: { color: '#22c55e', width: 2 },
        itemStyle: { color: '#22c55e' },
      },
    ],
  }
})
</script>

<template>
  <NCard title="Submission Trend" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <VChart :option="option" autoresize style="height: 280px" />
  </NCard>
</template>
