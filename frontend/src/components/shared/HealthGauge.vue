<script setup lang="ts">
import { computed } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { GaugeChart } from 'echarts/charts'
import { CanvasRenderer } from 'echarts/renderers'

use([GaugeChart, CanvasRenderer])

const props = defineProps<{
  value: number
  title?: string
}>()

function getColor(val: number): string {
  if (val >= 80) return '#22c55e'
  if (val >= 50) return '#f59e0b'
  return '#ef4444'
}

const option = computed(() => ({
  series: [
    {
      type: 'gauge',
      startAngle: 210,
      endAngle: -30,
      min: 0,
      max: 100,
      radius: '100%',
      progress: { show: true, width: 14 },
      axisLine: { lineStyle: { width: 14, color: [[1, '#e5e7eb']] } },
      axisTick: { show: false },
      splitLine: { show: false },
      axisLabel: { show: false },
      pointer: { show: false },
      title: {
        show: !!props.title,
        offsetCenter: [0, '70%'],
        fontSize: 13,
        color: '#888',
      },
      detail: {
        valueAnimation: true,
        fontSize: 28,
        fontWeight: 700,
        offsetCenter: [0, '20%'],
        formatter: '{value}',
        color: getColor(props.value),
      },
      itemStyle: { color: getColor(props.value) },
      data: [{ value: props.value, name: props.title || '' }],
    },
  ],
}))
</script>

<template>
  <VChart :option="option" autoresize style="height: 160px; width: 100%" />
</template>
