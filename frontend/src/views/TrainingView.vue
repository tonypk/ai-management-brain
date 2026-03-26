<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  NButton, NIcon, NSpin, NDataTable, NTag,
  NModal, NInput, NSelect, NSpace, NFormItem, NInputNumber, NSwitch,
  useMessage,
} from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import {
  listTrainingPrograms, createTrainingProgram, deleteTrainingProgram,
  listEnrollments, createEnrollment, updateEnrollment,
} from '@/api/training'
import { listEmployees } from '@/api/employees'
import type { TrainingProgram, TrainingEnrollment, Employee } from '@/types'

const message = useMessage()

const loading = ref(true)
const programs = ref<TrainingProgram[]>([])
const employees = ref<Employee[]>([])

// Program modal
const showProgramModal = ref(false)
const programForm = ref({
  title: '', description: '', category: '', duration_hours: 0,
  provider: '', url: '', is_mandatory: false,
})

// Enrollment detail
const selectedProgram = ref<TrainingProgram | null>(null)
const showEnrollmentModal = ref(false)
const enrollments = ref<TrainingEnrollment[]>([])
const enrollmentsLoading = ref(false)
const enrollForm = ref({ employee_id: '' })

onMounted(async () => {
  try {
    const [p, e] = await Promise.all([listTrainingPrograms(), listEmployees()])
    programs.value = p
    employees.value = e
  } catch {
    message.error('Failed to load data')
  } finally {
    loading.value = false
  }
})

async function handleCreateProgram() {
  try {
    const p = await createTrainingProgram(programForm.value)
    programs.value.unshift({ ...p, enrollment_count: 0, completed_count: 0 })
    showProgramModal.value = false
    programForm.value = {
      title: '', description: '', category: '', duration_hours: 0,
      provider: '', url: '', is_mandatory: false,
    }
    message.success('Program created')
  } catch {
    message.error('Failed to create program')
  }
}

async function handleDeleteProgram(id: string) {
  try {
    await deleteTrainingProgram(id)
    programs.value = programs.value.filter((p) => p.id !== id)
    message.success('Program deleted')
  } catch {
    message.error('Failed to delete program')
  }
}

async function openEnrollments(p: TrainingProgram) {
  selectedProgram.value = p
  showEnrollmentModal.value = true
  enrollmentsLoading.value = true
  try {
    enrollments.value = await listEnrollments(p.id)
  } catch {
    enrollments.value = []
  } finally {
    enrollmentsLoading.value = false
  }
}

async function handleEnroll() {
  if (!selectedProgram.value || !enrollForm.value.employee_id) return
  try {
    const e = await createEnrollment(selectedProgram.value.id, {
      employee_id: enrollForm.value.employee_id,
    })
    const emp = employees.value.find((x) => x.id === e.employee_id)
    enrollments.value.push({ ...e, employee_name: emp?.name ?? '' })
    enrollForm.value.employee_id = ''
    message.success('Employee enrolled')
  } catch {
    message.error('Failed to enroll')
  }
}

async function handleUpdateStatus(enrollment: TrainingEnrollment, newStatus: string) {
  if (!selectedProgram.value) return
  try {
    const updated = await updateEnrollment(selectedProgram.value.id, enrollment.id, {
      status: newStatus, score: enrollment.score, notes: enrollment.notes,
    })
    Object.assign(enrollment, updated)
    message.success('Status updated')
  } catch {
    message.error('Failed to update')
  }
}

const employeeOptions = computed(() =>
  employees.value.map((e) => ({ label: e.name, value: e.id })),
)

const statusColor = (s: string) => {
  if (s === 'completed') return 'success'
  if (s === 'in_progress') return 'warning'
  if (s === 'dropped') return 'error'
  return 'default'
}

const programColumns: DataTableColumns<TrainingProgram> = [
  { title: 'Title', key: 'title' },
  { title: 'Category', key: 'category', render: (p) => p.category || '—' },
  { title: 'Hours', key: 'duration_hours', width: 70 },
  {
    title: 'Mandatory', key: 'is_mandatory', width: 90,
    render: (p) => h(NTag, { type: p.is_mandatory ? 'error' : 'default', size: 'small' }, () => p.is_mandatory ? 'Yes' : 'No'),
  },
  {
    title: 'Enrolled', key: 'enrollment_count', width: 80,
    render: (p) => `${p.completed_count}/${p.enrollment_count}`,
  },
  {
    title: '', key: 'action', width: 130,
    render: (p) => h(NSpace, { size: 4 }, () => [
      h(NButton, { size: 'small', onClick: () => openEnrollments(p) }, () => 'View'),
      h(NButton, { size: 'tiny', type: 'error', quaternary: true, onClick: () => handleDeleteProgram(p.id) }, () => 'Del'),
    ]),
  },
]

const enrollmentStatusOptions = [
  { label: 'Enrolled', value: 'enrolled' },
  { label: 'In Progress', value: 'in_progress' },
  { label: 'Completed', value: 'completed' },
  { label: 'Dropped', value: 'dropped' },
]
</script>

<script lang="ts">
import { h } from 'vue'
export default {}
</script>

<template>
  <div>
    <PageHeader title="Training Programs">
      <template #actions>
        <NButton type="primary" @click="showProgramModal = true">
          <template #icon><NIcon :component="AddOutline" /></template>
          New Program
        </NButton>
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <EmptyState v-if="programs.length === 0 && !loading" description="No training programs yet" />
      <NDataTable v-else :columns="programColumns" :data="programs" :bordered="false" size="small" />
    </NSpin>

    <!-- Create Program Modal -->
    <NModal v-model:show="showProgramModal" preset="card" title="New Training Program" style="max-width: 500px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Title" :show-feedback="false">
          <NInput v-model:value="programForm.title" placeholder="e.g. Leadership Essentials" />
        </NFormItem>
        <NFormItem label="Category" :show-feedback="false">
          <NInput v-model:value="programForm.category" placeholder="e.g. Management, Technical" />
        </NFormItem>
        <NFormItem label="Description" :show-feedback="false">
          <NInput v-model:value="programForm.description" type="textarea" :rows="2" />
        </NFormItem>
        <NSpace :size="12">
          <NFormItem label="Duration (hours)" :show-feedback="false">
            <NInputNumber v-model:value="programForm.duration_hours" :min="0" style="width: 120px" />
          </NFormItem>
          <NFormItem label="Mandatory" :show-feedback="false">
            <NSwitch v-model:value="programForm.is_mandatory" />
          </NFormItem>
        </NSpace>
        <NFormItem label="Provider" :show-feedback="false">
          <NInput v-model:value="programForm.provider" placeholder="e.g. Coursera, Internal" />
        </NFormItem>
        <NFormItem label="URL" :show-feedback="false">
          <NInput v-model:value="programForm.url" placeholder="https://..." />
        </NFormItem>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showProgramModal = false">Cancel</NButton>
          <NButton type="primary" :disabled="!programForm.title.trim()" @click="handleCreateProgram">Create</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Enrollment Detail Modal -->
    <NModal v-model:show="showEnrollmentModal" preset="card" :title="selectedProgram?.title ?? 'Enrollments'" style="max-width: 560px; width: 95%">
      <NSpin :show="enrollmentsLoading">
        <div style="margin-bottom: 12px; display: flex; gap: 8px">
          <NSelect v-model:value="enrollForm.employee_id" :options="employeeOptions" placeholder="Select employee" style="flex: 1" />
          <NButton :disabled="!enrollForm.employee_id" @click="handleEnroll">Enroll</NButton>
        </div>

        <EmptyState v-if="enrollments.length === 0 && !enrollmentsLoading" description="No enrollments yet" />
        <div v-for="e in enrollments" :key="e.id" style="display: flex; align-items: center; gap: 8px; padding: 8px 0; border-bottom: 1px solid #f5f5f5">
          <div style="flex: 1; font-size: 13px">{{ e.employee_name }}</div>
          <NSelect
            :value="e.status"
            :options="enrollmentStatusOptions"
            size="small"
            style="width: 130px"
            @update:value="(v: string) => handleUpdateStatus(e, v)"
          />
          <NTag :type="statusColor(e.status)" size="small">{{ e.status }}</NTag>
        </div>
      </NSpin>
    </NModal>
  </div>
</template>
