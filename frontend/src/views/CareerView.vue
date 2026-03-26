<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  NButton, NIcon, NCard, NSpin, NDataTable, NTag,
  NModal, NInput, NSelect, NSpace, NFormItem, NInputNumber,
  useMessage,
} from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import {
  listCareerLevels, createCareerLevel, deleteCareerLevel,
  listCareerPaths, upsertCareerPath,
} from '@/api/career'
import { listEmployees } from '@/api/employees'
import type { CareerLevel, CareerPath, Employee } from '@/types'

const message = useMessage()

const loading = ref(true)
const levels = ref<CareerLevel[]>([])
const paths = ref<CareerPath[]>([])
const employees = ref<Employee[]>([])

// Level modal
const showLevelModal = ref(false)
const levelForm = ref({ title: '', level_order: 0, description: '', requirements: '' })

// Path modal
const showPathModal = ref(false)
const pathForm = ref({
  employee_id: '', current_level_id: '', target_level_id: '',
  target_date: '', notes: '',
})

onMounted(async () => {
  try {
    const [l, p, e] = await Promise.all([listCareerLevels(), listCareerPaths(), listEmployees()])
    levels.value = l
    paths.value = p
    employees.value = e
  } catch {
    message.error('Failed to load data')
  } finally {
    loading.value = false
  }
})

async function handleCreateLevel() {
  try {
    const l = await createCareerLevel(levelForm.value)
    levels.value.push(l)
    levels.value.sort((a, b) => a.level_order - b.level_order)
    showLevelModal.value = false
    levelForm.value = { title: '', level_order: 0, description: '', requirements: '' }
    message.success('Level created')
  } catch {
    message.error('Failed to create level')
  }
}

async function handleDeleteLevel(id: string) {
  try {
    await deleteCareerLevel(id)
    levels.value = levels.value.filter((l) => l.id !== id)
    message.success('Level deleted')
  } catch {
    message.error('Failed to delete level')
  }
}

async function handleUpsertPath() {
  try {
    await upsertCareerPath(pathForm.value)
    paths.value = await listCareerPaths()
    showPathModal.value = false
    message.success('Career path saved')
  } catch {
    message.error('Failed to save career path')
  }
}

const employeeOptions = computed(() =>
  employees.value.map((e) => ({ label: e.name, value: e.id })),
)

const levelOptions = computed(() =>
  levels.value.map((l) => ({ label: `${l.level_order}. ${l.title}`, value: l.id })),
)

const levelColumns: DataTableColumns<CareerLevel> = [
  { title: '#', key: 'level_order', width: 50 },
  { title: 'Title', key: 'title' },
  { title: 'Description', key: 'description', ellipsis: { tooltip: true } },
  {
    title: '', key: 'action', width: 60,
    render: (l) => h(NButton, { size: 'tiny', type: 'error', quaternary: true, onClick: () => handleDeleteLevel(l.id) }, () => 'Del'),
  },
]

const pathColumns: DataTableColumns<CareerPath> = [
  { title: 'Employee', key: 'employee_name' },
  {
    title: 'Current Level', key: 'current_level_title',
    render: (p) => p.current_level_title
      ? h(NTag, { size: 'small', type: 'info' }, () => p.current_level_title)
      : '—',
  },
  {
    title: 'Target Level', key: 'target_level_title',
    render: (p) => p.target_level_title
      ? h(NTag, { size: 'small', type: 'success' }, () => p.target_level_title)
      : '—',
  },
  { title: 'Target Date', key: 'target_date', width: 110, render: (p) => p.target_date ?? '—' },
  { title: 'Notes', key: 'notes', ellipsis: { tooltip: true } },
]
</script>

<script lang="ts">
import { h } from 'vue'
export default {}
</script>

<template>
  <div>
    <PageHeader title="Career Paths">
      <template #actions>
        <NSpace :size="8">
          <NButton @click="showPathModal = true; pathForm = { employee_id: '', current_level_id: '', target_level_id: '', target_date: '', notes: '' }">
            Assign Path
          </NButton>
          <NButton type="primary" @click="showLevelModal = true; levelForm = { title: '', level_order: levels.length, description: '', requirements: '' }">
            <template #icon><NIcon :component="AddOutline" /></template>
            New Level
          </NButton>
        </NSpace>
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <!-- Career Levels -->
      <NCard :bordered="false" size="small" style="margin-bottom: 20px">
        <div style="font-weight: 600; font-size: 14px; margin-bottom: 8px">Career Ladder ({{ levels.length }} levels)</div>
        <EmptyState v-if="levels.length === 0" description="No career levels defined yet" />
        <NDataTable v-else :columns="levelColumns" :data="levels" :bordered="false" size="small" />
      </NCard>

      <!-- Career Paths -->
      <NCard :bordered="false" size="small">
        <div style="font-weight: 600; font-size: 14px; margin-bottom: 8px">Employee Career Paths ({{ paths.length }})</div>
        <EmptyState v-if="paths.length === 0" description="No career paths assigned yet" />
        <NDataTable v-else :columns="pathColumns" :data="paths" :bordered="false" size="small" />
      </NCard>
    </NSpin>

    <!-- Create Level Modal -->
    <NModal v-model:show="showLevelModal" preset="card" title="New Career Level" style="max-width: 420px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Title" :show-feedback="false">
          <NInput v-model:value="levelForm.title" placeholder="e.g. Senior Engineer" />
        </NFormItem>
        <NFormItem label="Level Order" :show-feedback="false">
          <NInputNumber v-model:value="levelForm.level_order" :min="0" style="width: 100px" />
        </NFormItem>
        <NFormItem label="Description" :show-feedback="false">
          <NInput v-model:value="levelForm.description" type="textarea" :rows="2" />
        </NFormItem>
        <NFormItem label="Requirements" :show-feedback="false">
          <NInput v-model:value="levelForm.requirements" type="textarea" :rows="2" placeholder="What is needed to reach this level" />
        </NFormItem>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showLevelModal = false">Cancel</NButton>
          <NButton type="primary" :disabled="!levelForm.title.trim()" @click="handleCreateLevel">Create</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Assign Path Modal -->
    <NModal v-model:show="showPathModal" preset="card" title="Assign Career Path" style="max-width: 420px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Employee" :show-feedback="false">
          <NSelect v-model:value="pathForm.employee_id" :options="employeeOptions" placeholder="Select employee" />
        </NFormItem>
        <NFormItem label="Current Level" :show-feedback="false">
          <NSelect v-model:value="pathForm.current_level_id" :options="levelOptions" placeholder="Select current level" clearable />
        </NFormItem>
        <NFormItem label="Target Level" :show-feedback="false">
          <NSelect v-model:value="pathForm.target_level_id" :options="levelOptions" placeholder="Select target level" clearable />
        </NFormItem>
        <NFormItem label="Target Date" :show-feedback="false">
          <input v-model="pathForm.target_date" type="date" style="padding: 6px 10px; border: 1px solid #e0e0e6; border-radius: 3px; font-size: 14px" />
        </NFormItem>
        <NFormItem label="Notes" :show-feedback="false">
          <NInput v-model:value="pathForm.notes" type="textarea" :rows="2" />
        </NFormItem>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showPathModal = false">Cancel</NButton>
          <NButton type="primary" :disabled="!pathForm.employee_id" @click="handleUpsertPath">Save</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
