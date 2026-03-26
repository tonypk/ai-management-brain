<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  NSpin, NCard, NTag, NStatistic, NGrid, NGi,
  NDataTable, NTabs, NTabPane, NEmpty, NSpace,
  useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import { getCompanyState, listCommunicationEvents, listExecutionSignals } from '@/api/state'
import type { CompanyState, CommunicationEvent, ExecutionSignal } from '@/types'

const message = useMessage()

const loading = ref(true)
const state = ref<CompanyState | null>(null)
const events = ref<CommunicationEvent[]>([])
const signals = ref<ExecutionSignal[]>([])

onMounted(async () => {
  try {
    const [s, e, sig] = await Promise.all([
      getCompanyState(),
      listCommunicationEvents({ limit: 50 }),
      listExecutionSignals({ limit: 50 }),
    ])
    state.value = s
    events.value = e
    signals.value = sig
  } catch {
    message.error('Failed to load company state')
  } finally {
    loading.value = false
  }
})

function totalTasks(): number {
  return state.value?.task_stats?.reduce((sum, s) => sum + s.count, 0) ?? 0
}

function totalEvents(): number {
  return state.value?.event_counts?.reduce((sum, e) => sum + e.count, 0) ?? 0
}

const signalColumns: DataTableColumns<ExecutionSignal> = [
  { title: 'Type', key: 'signal_type', width: 120 },
  { title: 'Subject', key: 'subject_id', width: 100 },
  {
    title: 'Score', key: 'score', width: 80,
    render: (s) => h(NTag, {
      size: 'small',
      type: parseFloat(s.score) >= 0.7 ? 'error' : parseFloat(s.score) >= 0.4 ? 'warning' : 'success',
    }, () => s.score),
  },
  { title: 'Reasons', key: 'reasons', render: (s) => s.reasons?.join(', ') || '—' },
  { title: 'Window', key: 'time_window', width: 90 },
  { title: 'Generated', key: 'generated_at', width: 150, render: (s) => s.generated_at.slice(0, 16).replace('T', ' ') },
]

const eventColumns: DataTableColumns<CommunicationEvent> = [
  { title: 'Type', key: 'event_type', width: 110 },
  { title: 'Platform', key: 'platform', width: 90 },
  { title: 'Actor', key: 'actor_name', width: 100, render: (e) => e.actor_name || '—' },
  { title: 'Target', key: 'target_name', width: 100, render: (e) => e.target_name || '—' },
  {
    title: 'Confidence', key: 'confidence', width: 90,
    render: (e) => h(NTag, { size: 'small' }, () => e.confidence),
  },
  { title: 'Time', key: 'occurred_at', width: 150, render: (e) => e.occurred_at.slice(0, 16).replace('T', ' ') },
]
</script>

<script lang="ts">
import { h } from 'vue'
export default {}
</script>

<template>
  <div>
    <PageHeader title="Company State" />

    <NSpin :show="loading">
      <template v-if="state">
        <!-- Summary Stats -->
        <NGrid :cols="4" :x-gap="12" style="margin-bottom: 16px">
          <NGi>
            <NCard :bordered="false" size="small">
              <NStatistic label="Top Risks" :value="state.top_risks?.length ?? 0" />
            </NCard>
          </NGi>
          <NGi>
            <NCard :bordered="false" size="small">
              <NStatistic label="Overdue Tasks" :value="state.overdue_tasks?.length ?? 0" />
            </NCard>
          </NGi>
          <NGi>
            <NCard :bordered="false" size="small">
              <NStatistic label="Total Tasks" :value="totalTasks()" />
            </NCard>
          </NGi>
          <NGi>
            <NCard :bordered="false" size="small">
              <NStatistic label="Events" :value="totalEvents()" />
            </NCard>
          </NGi>
        </NGrid>

        <!-- Task Stats Breakdown -->
        <NCard :bordered="false" size="small" style="margin-bottom: 16px">
          <div style="font-weight: 600; font-size: 14px; margin-bottom: 8px">Task Status Breakdown</div>
          <NSpace :size="8">
            <NTag v-for="ts in state.task_stats" :key="ts.status" size="small">
              {{ ts.status }}: {{ ts.count }}
            </NTag>
          </NSpace>
          <div v-if="!state.task_stats?.length" style="color: #999; font-size: 13px">No task data</div>
        </NCard>

        <!-- Working Memory -->
        <NCard v-if="state.working_memory?.content" :bordered="false" size="small" style="margin-bottom: 16px">
          <div style="font-weight: 600; font-size: 14px; margin-bottom: 8px">Working Memory</div>
          <div style="font-size: 12px; color: #666; margin-bottom: 4px">
            Type: {{ state.working_memory.snapshot_type }} | By: {{ state.working_memory.generated_by }}
          </div>
          <pre style="font-size: 12px; white-space: pre-wrap; background: #f9f9f9; padding: 8px; border-radius: 4px">{{ JSON.stringify(state.working_memory.content, null, 2) }}</pre>
        </NCard>
      </template>

      <!-- Tabs for Events & Signals -->
      <NTabs type="line">
        <NTabPane name="signals" tab="Execution Signals">
          <NEmpty v-if="signals.length === 0" description="No execution signals" />
          <NDataTable v-else :columns="signalColumns" :data="signals" :bordered="false" size="small" />
        </NTabPane>
        <NTabPane name="events" tab="Communication Events">
          <NEmpty v-if="events.length === 0" description="No communication events" />
          <NDataTable v-else :columns="eventColumns" :data="events" :bordered="false" size="small" />
        </NTabPane>
      </NTabs>
    </NSpin>
  </div>
</template>
