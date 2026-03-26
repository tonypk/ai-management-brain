<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  NButton, NIcon, NSpin, NDataTable, NTag,
  NModal, NInput, NSelect, NSpace, NFormItem,
  useMessage,
} from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import { listProjects, createProject, deleteProject } from '@/api/projects'
import { listEmployees } from '@/api/employees'
import type { Project, Employee } from '@/types'

const message = useMessage()

const loading = ref(true)
const projects = ref<Project[]>([])
const employees = ref<Employee[]>([])

const showCreateModal = ref(false)
const form = ref({
  name: '', description: '', owner_id: '', status: 'planning',
  priority: 'medium', start_date: '', due_date: '',
})

onMounted(async () => {
  try {
    const [p, e] = await Promise.all([listProjects(), listEmployees()])
    projects.value = p
    employees.value = e
  } catch {
    message.error('Failed to load projects')
  } finally {
    loading.value = false
  }
})

async function handleCreate() {
  try {
    const p = await createProject(form.value)
    projects.value.unshift(p)
    showCreateModal.value = false
    form.value = { name: '', description: '', owner_id: '', status: 'planning', priority: 'medium', start_date: '', due_date: '' }
    message.success('Project created')
  } catch {
    message.error('Failed to create project')
  }
}

async function handleDelete(id: string) {
  try {
    await deleteProject(id)
    projects.value = projects.value.filter((p) => p.id !== id)
    message.success('Project deleted')
  } catch {
    message.error('Failed to delete project')
  }
}

const employeeOptions = computed(() =>
  employees.value.map((e) => ({ label: e.name, value: e.id })),
)

const statusOptions = [
  { label: 'Planning', value: 'planning' },
  { label: 'Active', value: 'active' },
  { label: 'On Hold', value: 'on_hold' },
  { label: 'Completed', value: 'completed' },
  { label: 'Cancelled', value: 'cancelled' },
]

const priorityOptions = [
  { label: 'Critical', value: 'critical' },
  { label: 'High', value: 'high' },
  { label: 'Medium', value: 'medium' },
  { label: 'Low', value: 'low' },
]

const statusColor = (s: string) => {
  if (s === 'active') return 'success'
  if (s === 'planning') return 'info'
  if (s === 'on_hold') return 'warning'
  if (s === 'completed') return 'default'
  if (s === 'cancelled') return 'error'
  return 'default'
}

const priorityColor = (p: string) => {
  if (p === 'critical') return 'error'
  if (p === 'high') return 'warning'
  if (p === 'medium') return 'info'
  return 'default'
}

const columns: DataTableColumns<Project> = [
  { title: 'Name', key: 'name' },
  {
    title: 'Status', key: 'status', width: 100,
    render: (p) => h(NTag, { size: 'small', type: statusColor(p.status) }, () => p.status),
  },
  {
    title: 'Priority', key: 'priority', width: 90,
    render: (p) => h(NTag, { size: 'small', type: priorityColor(p.priority) }, () => p.priority),
  },
  { title: 'Owner', key: 'owner_name', width: 120, render: (p) => p.owner_name || '—' },
  { title: 'Due', key: 'due_date', width: 110, render: (p) => p.due_date ?? '—' },
  {
    title: 'Blockers', key: 'blockers', width: 80,
    render: (p) => p.blockers?.length
      ? h(NTag, { size: 'small', type: 'error' }, () => `${p.blockers.length}`)
      : '—',
  },
  {
    title: '', key: 'action', width: 60,
    render: (p) => h(NButton, { size: 'tiny', type: 'error', quaternary: true, onClick: () => handleDelete(p.id) }, () => 'Del'),
  },
]
</script>

<script lang="ts">
import { h } from 'vue'
export default {}
</script>

<template>
  <div>
    <PageHeader title="Projects">
      <template #actions>
        <NButton type="primary" @click="showCreateModal = true">
          <template #icon><NIcon :component="AddOutline" /></template>
          New Project
        </NButton>
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <EmptyState v-if="projects.length === 0 && !loading" description="No projects yet" />
      <NDataTable v-else :columns="columns" :data="projects" :bordered="false" size="small" />
    </NSpin>

    <!-- Create Project Modal -->
    <NModal v-model:show="showCreateModal" preset="card" title="New Project" style="max-width: 500px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Name" :show-feedback="false">
          <NInput v-model:value="form.name" placeholder="e.g. Q2 Product Launch" />
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
        <NFormItem label="Owner" :show-feedback="false">
          <NSelect v-model:value="form.owner_id" :options="employeeOptions" placeholder="Select owner" clearable />
        </NFormItem>
        <NSpace :size="12">
          <NFormItem label="Start Date" :show-feedback="false">
            <input v-model="form.start_date" type="date" style="padding: 6px 10px; border: 1px solid #e0e0e6; border-radius: 3px; font-size: 14px" />
          </NFormItem>
          <NFormItem label="Due Date" :show-feedback="false">
            <input v-model="form.due_date" type="date" style="padding: 6px 10px; border: 1px solid #e0e0e6; border-radius: 3px; font-size: 14px" />
          </NFormItem>
        </NSpace>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCreateModal = false">Cancel</NButton>
          <NButton type="primary" :disabled="!form.name.trim()" @click="handleCreate">Create</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
