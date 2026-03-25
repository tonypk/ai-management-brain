<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent, MarkLineComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { NSelect, NText } from 'naive-ui'
import { listSnapshots } from '@/api/goals'
import type { Objective, GoalSnapshot } from '@/types'

use([LineChart, GridComponent, TooltipComponent, LegendComponent, MarkLineComponent, CanvasRenderer])

const props = defineProps<{
  objectives: Objective[]
}>()

const selectedGoalId = ref<string | null>(null)
const snapshots = ref<GoalSnapshot[]>([])
const loading = ref(false)

const goalOptions = computed(() =>
  props.objectives.map((o) => ({ label: o.title, value: o.id }))
)

const selectedGoal = computed(() =>
  props.objectives.find((o) => o.id === selectedGoalId.value) ?? null
)

// Quarter date helpers
function quarterStartDate(cycle: string): Date {
  const [year, qPart] = cycle.split('-Q')
  const q = parseInt(qPart, 10)
  return new Date(parseInt(year, 10), (q - 1) * 3, 1)
}

function quarterEndDate(cycle: string): Date {
  const [year, qPart] = cycle.split('-Q')
  const q = parseInt(qPart, 10)
  return new Date(parseInt(year, 10), q * 3, 0) // last day of quarter
}

function formatDate(d: Date): string {
  return d.toISOString().slice(0, 10)
}

// Expected linear progress: 0% at quarter start → 100% at quarter end
function expectedProgress(date: string, cycle: string): number {
  const start = quarterStartDate(cycle).getTime()
  const end = quarterEndDate(cycle).getTime()
  const current = new Date(date).getTime()
  if (current <= start) return 0
  if (current >= end) return 100
  return Math.round(((current - start) / (end - start)) * 100)
}

async function fetchSnapshots() {
  if (!selectedGoalId.value) {
    snapshots.value = []
    return
  }
  loading.value = true
  try {
    snapshots.value = await listSnapshots(selectedGoalId.value)
  } catch {
    snapshots.value = []
  } finally {
    loading.value = false
  }
}

watch(selectedGoalId, fetchSnapshots)

// Auto-select first goal
watch(
  () => props.objectives,
  (objs) => {
    if (objs.length > 0 && !selectedGoalId.value) {
      selectedGoalId.value = objs[0].id
    }
  },
  { immediate: true }
)

const option = computed(() => {
  const goal = selectedGoal.value
  if (!goal || snapshots.value.length === 0) return null

  const cycle = goal.cycle
  const start = quarterStartDate(cycle)
  const end = quarterEndDate(cycle)

  // Build date axis — from quarter start to today or quarter end
  const today = new Date()
  const axisEnd = today < end ? today : end

  // Actual progress from snapshots (sorted by date)
  const sorted = [...snapshots.value].sort(
    (a, b) => new Date(a.snapshot_date).getTime() - new Date(b.snapshot_date).getTime()
  )
  const actualDates = sorted.map((s) => s.snapshot_date)
  const actualValues = sorted.map((s) => Math.round(s.overall_progress * 100) / 100)

  // Expected linear line: build points matching actual dates + start/end
  const expectedDates = [formatDate(start), ...actualDates]
  if (formatDate(axisEnd) !== expectedDates[expectedDates.length - 1]) {
    expectedDates.push(formatDate(axisEnd))
  }
  const uniqueDates = [...new Set(expectedDates)].sort()
  const expectedValues = uniqueDates.map((d) => expectedProgress(d, cycle))

  // Current overall progress
  const currentProgress = actualValues.length > 0 ? actualValues[actualValues.length - 1] : 0
  const todayExpected = expectedProgress(formatDate(today), cycle)
  const deviation = Math.round((currentProgress - todayExpected) * 100) / 100

  return {
    tooltip: {
      trigger: 'axis' as const,
      formatter: (params: Array<{ seriesName: string; value: number; axisValue: string }>) => {
        let tip = params[0]?.axisValue ?? ''
        for (const p of params) {
          tip += `<br/>${p.seriesName}: ${p.value}%`
        }
        return tip
      },
    },
    legend: { data: ['Actual', 'Expected'], bottom: 0 },
    grid: { left: 50, right: 30, top: 30, bottom: 40 },
    xAxis: {
      type: 'category' as const,
      data: uniqueDates,
      axisLabel: { rotate: 30, fontSize: 11 },
    },
    yAxis: {
      type: 'value' as const,
      max: 100,
      axisLabel: { formatter: '{value}%' },
    },
    series: [
      {
        name: 'Actual',
        type: 'line' as const,
        data: uniqueDates.map((d) => {
          const idx = actualDates.indexOf(d)
          return idx >= 0 ? actualValues[idx] : null
        }),
        smooth: true,
        lineStyle: { width: 3, color: deviation >= 0 ? '#22c55e' : '#ef4444' },
        itemStyle: { color: deviation >= 0 ? '#22c55e' : '#ef4444' },
        connectNulls: true,
      },
      {
        name: 'Expected',
        type: 'line' as const,
        data: expectedValues,
        lineStyle: { width: 2, type: 'dashed' as const, color: '#94a3b8' },
        itemStyle: { color: '#94a3b8' },
        symbol: 'none',
      },
    ],
  }
})

const deviationInfo = computed(() => {
  const goal = selectedGoal.value
  if (!goal || snapshots.value.length === 0) return null
  const sorted = [...snapshots.value].sort(
    (a, b) => new Date(a.snapshot_date).getTime() - new Date(b.snapshot_date).getTime()
  )
  const currentProgress = sorted[sorted.length - 1].overall_progress
  const todayExpected = expectedProgress(formatDate(new Date()), goal.cycle)
  const deviation = Math.round((currentProgress - todayExpected) * 100) / 100
  return { currentProgress: Math.round(currentProgress * 100) / 100, todayExpected, deviation }
})
</script>

<template>
  <div>
    <div style="display: flex; align-items: center; gap: 12px; margin-bottom: 12px">
      <NText style="font-size: 13px; white-space: nowrap">Goal:</NText>
      <NSelect
        v-model:value="selectedGoalId"
        :options="goalOptions"
        size="small"
        style="max-width: 300px"
        placeholder="Select a goal"
      />
    </div>

    <div v-if="deviationInfo" style="display: flex; gap: 16px; margin-bottom: 8px; font-size: 13px">
      <span>Actual: <strong>{{ deviationInfo.currentProgress }}%</strong></span>
      <span>Expected: <strong>{{ deviationInfo.todayExpected }}%</strong></span>
      <span :style="{ color: deviationInfo.deviation >= 0 ? '#22c55e' : '#ef4444', fontWeight: 600 }">
        {{ deviationInfo.deviation >= 0 ? '+' : '' }}{{ deviationInfo.deviation }}% deviation
      </span>
    </div>

    <VChart
      v-if="option"
      :option="option"
      :loading="loading"
      autoresize
      style="height: 280px; width: 100%"
    />
    <div v-else-if="selectedGoalId && !loading" style="color: #999; padding: 20px; text-align: center; font-size: 13px">
      No snapshot data yet. Snapshots are recorded daily.
    </div>
    <div v-else-if="!selectedGoalId" style="color: #999; padding: 20px; text-align: center; font-size: 13px">
      Select a goal to view deviation tracking
    </div>
  </div>
</template>
