<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NButton, NInput, NSpace, useMessage, type DataTableColumn } from 'naive-ui'
import { listSchedulerJobs, updateJobSchedule, triggerJob } from '@/api/settings'
import type { SchedulerJob } from '@/types'

const message = useMessage()
const loading = ref(false)
const jobs = ref<SchedulerJob[]>([])
const editingJob = ref<string | null>(null)
const editCron = ref('')

onMounted(loadJobs)

async function loadJobs(): Promise<void> {
  loading.value = true
  try {
    jobs.value = await listSchedulerJobs()
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to load jobs')
  } finally {
    loading.value = false
  }
}

function startEdit(job: SchedulerJob): void {
  editingJob.value = job.name
  editCron.value = job.cron
}

async function saveEdit(jobName: string): Promise<void> {
  try {
    await updateJobSchedule(jobName, editCron.value)
    editingJob.value = null
    message.success('Schedule updated')
    await loadJobs()
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to update')
  }
}

async function handleTrigger(jobName: string): Promise<void> {
  try {
    await triggerJob(jobName)
    message.success(`Job "${jobName}" triggered`)
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to trigger')
  }
}

const columns: DataTableColumn<SchedulerJob>[] = [
  { title: 'Job', key: 'name' },
  {
    title: 'Schedule',
    key: 'cron',
    render(row) {
      if (editingJob.value === row.name) {
        return h(NSpace, { size: 4 }, () => [
          h(NInput, { value: editCron.value, size: 'small', style: 'width: 140px', 'onUpdate:value': (v: string) => { editCron.value = v } }),
          h(NButton, { size: 'small', type: 'primary', onClick: () => saveEdit(row.name) }, () => 'Save'),
          h(NButton, { size: 'small', onClick: () => { editingJob.value = null } }, () => 'Cancel'),
        ])
      }
      return h('span', { style: 'cursor: pointer', onClick: () => startEdit(row) }, row.cron)
    },
  },
  { title: 'Last Run', key: 'last_run' },
  { title: 'Next Run', key: 'next_run' },
  {
    title: 'Action',
    key: 'action',
    render(row) {
      return h(NButton, { size: 'small', type: 'primary', ghost: true, onClick: () => handleTrigger(row.name) }, () => 'Run Now')
    },
  },
]
</script>

<template>
  <NCard title="Scheduler" :bordered="false">
    <NDataTable :columns="columns" :data="jobs" :loading="loading" :bordered="false" size="small" />
  </NCard>
</template>
