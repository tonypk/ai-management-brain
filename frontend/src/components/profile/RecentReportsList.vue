<script setup lang="ts">
import { NCard, NEmpty } from 'naive-ui'
import ReportCard from '@/components/reports/ReportCard.vue'
import type { Report } from '@/types'

defineProps<{
  reports: Report[]
}>()
</script>

<template>
  <NCard title="Recent Reports (7 days)" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <template v-if="reports.length">
      <ReportCard v-for="report in reports" :key="report.id" :report="report" />
    </template>
    <NEmpty v-else description="No reports in the last 7 days" />
  </NCard>
</template>
