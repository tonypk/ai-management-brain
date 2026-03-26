<script setup lang="ts">
import { computed } from 'vue'
import { NCard } from 'naive-ui'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { PieChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

use([PieChart, TooltipComponent, LegendComponent, CanvasRenderer])

const props = defineProps<{
  data: Record<string, number>
}>()

const colorMap: Record<string, string> = {
  positive: '#22c55e',
  neutral: '#6366f1',
  negative: '#ef4444',
  mixed: '#f59e0b',
}

const option = computed(() => {
  const items = Object.entries(props.data).map(([name, value]) => ({
    name,
    value,
    itemStyle: { color: colorMap[name] || '#94a3b8' },
  }))

  return {
    tooltip: {
      trigger: 'item' as const,
      formatter: '{b}: {c} ({d}%)',
    },
    legend: {
      bottom: 0,
      data: items.map(i => i.name),
    },
    series: [
      {
        type: 'pie' as const,
        radius: ['40%', '70%'],
        center: ['50%', '45%'],
        avoidLabelOverlap: true,
        itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
        label: {
          show: true,
          formatter: '{b}\n{d}%',
          fontSize: 12,
        },
        data: items,
      },
    ],
  }
})
</script>

<template>
  <NCard title="Sentiment Distribution" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <VChart :option="option" autoresize style="height: 280px" />
  </NCard>
</template>
