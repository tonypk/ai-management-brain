<script setup lang="ts">
import { h } from 'vue'
import { NCard, NDataTable, type DataTableColumn } from 'naive-ui'
import SentimentBadge from '@/components/shared/SentimentBadge.vue'
import type { EmployeeActivity } from '@/types'

const props = defineProps<{
  data: EmployeeActivity[]
}>()

const columns: DataTableColumn<EmployeeActivity>[] = [
  { title: 'Name', key: 'name', sorter: 'default' },
  { title: '7d Submitted', key: 'submitted_7d', sorter: (a, b) => a.submitted_7d - b.submitted_7d },
  {
    title: '7d Missed',
    key: 'missed_7d',
    sorter: (a, b) => a.missed_7d - b.missed_7d,
    render(row) {
      const style = row.missed_7d >= 3 ? 'color: #ef4444; font-weight: 600' : ''
      return h('span', { style }, String(row.missed_7d))
    },
  },
  {
    title: 'Sentiment',
    key: 'last_sentiment',
    render(row) {
      return h(SentimentBadge, { sentiment: row.last_sentiment })
    },
    filterOptions: [
      { label: 'Positive', value: 'positive' },
      { label: 'Neutral', value: 'neutral' },
      { label: 'Mixed', value: 'mixed' },
      { label: 'Negative', value: 'negative' },
    ],
    filter(value, row) {
      return row.last_sentiment === value.toString()
    },
  },
  { title: 'Culture', key: 'culture_code', sorter: 'default' },
]
</script>

<template>
  <NCard title="Employee Activity" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <NDataTable
      :columns="columns"
      :data="data"
      :pagination="{ pageSize: 10 }"
      :bordered="false"
      size="small"
    />
  </NCard>
</template>
