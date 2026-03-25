<script setup lang="ts">
import { ref, watch } from 'vue'
import { NSpin } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import ReportCalendar from '@/components/reports/ReportCalendar.vue'
import DailySummaryCard from '@/components/reports/DailySummaryCard.vue'
import ReportCard from '@/components/reports/ReportCard.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import { listReports, getSummary } from '@/api/reports'
import type { Report, Summary } from '@/types'

const loading = ref(false)
const selectedDate = ref(Date.now())
const reports = ref<Report[]>([])
const summary = ref<Summary | null>(null)

function formatDate(ts: number): string {
  const d = new Date(ts)
  return d.toISOString().slice(0, 10)
}

async function loadData(): Promise<void> {
  loading.value = true
  const date = formatDate(selectedDate.value)
  try {
    const [r, s] = await Promise.all([
      listReports(date),
      getSummary(date).catch(() => null),
    ])
    reports.value = r
    summary.value = s
  } catch {
    reports.value = []
    summary.value = null
  } finally {
    loading.value = false
  }
}

watch(selectedDate, loadData, { immediate: true })
</script>

<template>
  <div>
    <PageHeader title="Reports" :breadcrumbs="[{ label: 'Dashboard', to: '/' }, { label: 'Reports' }]">
      <template #actions>
        <ReportCalendar v-model="selectedDate" />
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <DailySummaryCard :summary="summary" />

      <div style="margin-top: 16px">
        <template v-if="reports.length > 0">
          <ReportCard v-for="r in reports" :key="r.id" :report="r" />
        </template>
        <EmptyState v-else description="No reports for this date" />
      </div>
    </NSpin>
  </div>
</template>
