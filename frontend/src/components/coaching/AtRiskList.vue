<script setup lang="ts">
import { h } from 'vue'
import { NCard, NDataTable, type DataTableColumns } from 'naive-ui'
import { useRouter } from 'vue-router'
import PriorityBadge from './PriorityBadge.vue'
import SentimentBadge from '@/components/shared/SentimentBadge.vue'
import type { AtRiskEmployee } from '@/types'

const props = defineProps<{
  employees: AtRiskEmployee[]
  selectedId?: string
}>()

const emit = defineEmits<{
  select: [employee: AtRiskEmployee]
}>()

const router = useRouter()

const columns: DataTableColumns<AtRiskEmployee> = [
  {
    title: 'Name',
    key: 'name',
    render(row) {
      return h(
        'a',
        {
          style: 'color: #6366f1; cursor: pointer; text-decoration: none',
          onClick: (e: Event) => {
            e.stopPropagation()
            router.push(`/employees/${row.id}`)
          },
        },
        row.name,
      )
    },
  },
  {
    title: 'Risk',
    key: 'risk',
    width: 80,
    render(row) {
      return h(PriorityBadge, { level: row.risk })
    },
  },
  {
    title: 'Missed',
    key: 'missed_7d',
    width: 80,
  },
  {
    title: 'Sentiment',
    key: 'last_sentiment',
    width: 100,
    render(row) {
      return h(SentimentBadge, { sentiment: row.last_sentiment })
    },
  },
]

function handleRowClick(row: AtRiskEmployee): void {
  emit('select', row)
}

function rowProps(row: AtRiskEmployee) {
  return {
    style: row.id === props.selectedId ? 'background: #f0f0ff; cursor: pointer' : 'cursor: pointer',
    onClick: () => handleRowClick(row),
  }
}
</script>

<template>
  <NCard title="At-Risk Employees" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <NDataTable
      :columns="columns"
      :data="employees"
      :row-props="rowProps"
      :bordered="false"
      size="small"
      :pagination="{ pageSize: 10 }"
    />
  </NCard>
</template>
