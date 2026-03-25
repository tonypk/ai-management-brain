<script setup lang="ts">
import { NCard, NStatistic, NGrid, NGi } from 'naive-ui'
import type { Summary } from '@/types'

defineProps<{
  summary: Summary | null
}>()
</script>

<template>
  <NCard title="Daily Summary" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <template v-if="summary">
      <NGrid :cols="3" :x-gap="16" style="margin-bottom: 16px">
        <NGi>
          <NStatistic label="Submission Rate">
            {{ Math.round(summary.submission_rate * 100) }}%
          </NStatistic>
        </NGi>
        <NGi>
          <NStatistic label="Blockers" :value="summary.blockers_count" />
        </NGi>
        <NGi>
          <NStatistic label="Date" :value="summary.summary_date" />
        </NGi>
      </NGrid>
      <div style="background: #f8fafc; border-radius: 8px; padding: 16px; line-height: 1.7; white-space: pre-wrap; font-size: 14px">
        {{ summary.content }}
      </div>
    </template>
    <div v-else style="text-align: center; color: #888; padding: 24px">
      No summary available for this date.
    </div>
  </NCard>
</template>
