<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  NButton, NIcon, NSpin, NDataTable, NTag, NCard, NStatistic, NGrid, NGi,
  NModal, NInput, NSelect, NSpace, NFormItem,
  useMessage,
} from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import { listTasks, createTask, deleteTask, getTaskStats } from '@/api/tasks'
import { listProjects } from '@/api/projects'
import { listEmployees } from '@/api/employees'
import type { Task, TaskStats, Project, Employee } from '@/types'

const message = useMessage()

const loading = ref(true)
const tasks = ref<Task[]>([])
const stats = ref<TaskStats[]>([])
const projects = ref<Project[]>([])
const employees = ref<Employee[]>([])

const showCreateModal = ref(false)
const form = ref({
  title: '', description: '', project_id: '', owner_id: '',
  status: 'todo', priority: 'medium', due_at: '',
})

onMounted(async () => {
  try {
    const [t, s, p, e] = await Promise.all([
      listTasks(), getTaskStats(), listProjects(), listEmployees(),
    ])
    tasks.value = t
    stats.value = s
    projects.value = p
    employees.value = e
  } catch {
    message.error('Failed to load tasks')
  } finally {
    loading.value = false
  }
})

async function handleCreate() {
  try {
    const t = await createTask(form.value)
    tasks.value.unshift(t)
    stats.value = await getTaskStats()
    showCreateModal.value = false
    form.value = { title: '', description: '', project_id: '', owner_id: '', status: 'todo', priority: 'medium', due_at: '' }
    message.success('Task created')
  } catch {
    message.error('Failed to create task')
  }
}

async function handleDelete(id: string) {
  try {
    await deleteTask(id)
    tasks.value = tasks.value.filter((t) => t.id !== id)
    stats.value = await getTaskStats()
    message.success('Task deleted')
  } catch {
    message.error('Failed to delete task')
  }
}

const employeeOptions = computed(() =>
  employees.value.map((e) => ({ label: e.name, value: e.id })),
)

const projectOptions = computed(() =>
  projects.value.map((p) => ({ label: p.name, value: p.id })),
)

const statusOptions = [
  { label: 'To Do', value: 'todo' },
  { label: 'In Progress', value: 'in_progress' },
  { label: 'In Review', value: 'in_review' },
  { label: 'Done', value: 'done' },
  { label: 'Blocked', value: 'blocked' },
]

const priorityOptions = [
  { label: 'Critical', value: 'critical' },
  { label: 'High', value: 'high' },
  { label: 'Medium', value: 'medium' },
  { label: 'Low', value: 'low' },
]

const statusColor = (s: string) => {
  if (s === 'done') return 'success'
  if (s === 'in_progress') return 'warning'
  if (s === 'in_review') return 'info'
  if (s === 'blocked') return 'error'
  return 'default'
}

const priorityColor = (p: string) => {
  if (p === 'critical') return 'error'
  if (p === 'high') return 'warning'
  if (p === 'medium') return 'info'
  return 'default'
}

function getStatCount(status: string): number {
  return stats.value.find((s) => s.status === status)?.count ?? 0
}

const columns: DataTableColumns<Task> = [
  { title: 'Title', key: 'title' },
  {
    title: 'Status', key: 'status', width: 100,
    render: (t) => h(NTag, { size: 'small', type: statusColor(t.status) }, () => t.status),
  },
  {
    title: 'Priority', key: 'priority', width: 90,
    render: (t) => h(NTag, { size: 'small', type: priorityColor(t.priority) }, () => t.priority),
  },
  { title: 'Owner', key: 'owner_name', width: 120, render: (t) => t.owner_name || '—' },
  { title: 'Project', key: 'project_name', width: 130, render: (t) => t.project_name || '—' },
  { title: 'Due', key: 'due_at', width: 110, render: (t) => t.due_at ? t.due_at.slice(0, 10) : '—' },
  {
    title: '', key: 'action', width: 60,
    render: (t) => h(NButton, { size: 'tiny', type: 'error', quaternary: true, onClick: () => handleDelete(t.id) }, () => 'Del'),
  },
]
</script>

<script lang="ts">
import { h } from 'vue'
export default {}
</script>

<template>
  <div>
    <PageHeader title="Tasks">
      <template #actions>
        <NButton type="primary" @click="showCreateModal = true">
          <template #icon><NIcon :component="AddOutline" /></template>
          New Task
        </NButton>
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <!-- Stats Row -->
      <NGrid :cols="5" :x-gap="12" style="margin-bottom: 16px">
        <NGi>
          <NCard :bordered="false" size="small">
            <NStatistic label="To Do" :value="getStatCount('todo')" />
          </NCard>
        </NGi>
        <NGi>
          <NCard :bordered="false" size="small">
            <NStatistic label="In Progress" :value="getStatCount('in_progress')" />
          </NCard>
        </NGi>
        <NGi>
          <NCard :bordered="false" size="small">
            <NStatistic label="In Review" :value="getStatCount('in_review')" />
          </NCard>
        </NGi>
        <NGi>
          <NCard :bordered="false" size="small">
            <NStatistic label="Done" :value="getStatCount('done')" />
          </NCard>
        </NGi>
        <NGi>
          <NCard :bordered="false" size="small">
            <NStatistic label="Blocked" :value="getStatCount('blocked')" />
          </NCard>
        </NGi>
      </NGrid>

      <EmptyState v-if="tasks.length === 0 && !loading" description="No tasks yet" />
      <NDataTable v-else :columns="columns" :data="tasks" :bordered="false" size="small" />
    </NSpin>

    <!-- Create Task Modal -->
    <NModal v-model:show="showCreateModal" preset="card" title="New Task" style="max-width: 500px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Title" :show-feedback="false">
          <NInput v-model:value="form.title" placeholder="e.g. Implement user onboarding flow" />
        </NFormItem>
        <NFormItem label="Description" :show-feedback="false">
          <NInput v-model:value="form.description" type="textarea" :rows="2" />
        </NFormItem>
        <NSpace :size="12">
          <NFormItem label="Status" :show-feedback="false">
            <NSelect v-model:value="form.status" :options="statusOptions" style="width: 130px" />
          </NFormItem>
          <NFormItem label="Priority" :show-feedback="false">
            <NSelect v-model:value="form.priority" :options="priorityOptions" style="width: 120px" />
          </NFormItem>
        </NSpace>
        <NFormItem label="Project" :show-feedback="false">
          <NSelect v-model:value="form.project_id" :options="projectOptions" placeholder="Select project" clearable />
        </NFormItem>
        <NFormItem label="Owner" :show-feedback="false">
          <NSelect v-model:value="form.owner_id" :options="employeeOptions" placeholder="Select owner" clearable />
        </NFormItem>
        <NFormItem label="Due Date" :show-feedback="false">
          <input v-model="form.due_at" type="date" style="padding: 6px 10px; border: 1px solid #e0e0e6; border-radius: 3px; font-size: 14px" />
        </NFormItem>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCreateModal = false">Cancel</NButton>
          <NButton type="primary" :disabled="!form.title.trim()" @click="handleCreate">Create</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
