<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  NButton, NIcon, NCard, NSpin, NDataTable, NTag,
  NModal, NInput, NSelect, NSpace, NFormItem, NRate,
  useMessage,
} from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import {
  listReviewCycles, createReviewCycle,
  listReviews, createReview, updateReview,
} from '@/api/reviews'
import { listEmployees } from '@/api/employees'
import type { ReviewCycle, PerformanceReview, Employee } from '@/types'

const message = useMessage()

const loading = ref(true)
const cycles = ref<ReviewCycle[]>([])
const selectedCycleId = ref<string | null>(null)
const reviews = ref<PerformanceReview[]>([])
const employees = ref<Employee[]>([])
const reviewsLoading = ref(false)

// Cycle modal
const showCycleModal = ref(false)
const cycleForm = ref({ title: '', period: '', start_date: '', end_date: '' })

// Review modal
const showReviewModal = ref(false)
const reviewForm = ref({ employee_id: '', reviewer_id: '' })

// Review detail modal
const showDetailModal = ref(false)
const editingReview = ref<PerformanceReview | null>(null)
const detailForm = ref({
  status: 'pending',
  self_rating: undefined as number | undefined,
  manager_rating: undefined as number | undefined,
  self_summary: '',
  manager_summary: '',
  strengths: '',
  improvements: '',
})

onMounted(async () => {
  try {
    const [c, e] = await Promise.all([listReviewCycles(), listEmployees()])
    cycles.value = c
    employees.value = e
    if (c.length > 0) {
      selectedCycleId.value = c[0].id
      await loadReviews(c[0].id)
    }
  } catch {
    message.error('Failed to load data')
  } finally {
    loading.value = false
  }
})

async function loadReviews(cycleId: string) {
  reviewsLoading.value = true
  try {
    reviews.value = await listReviews(cycleId)
  } catch {
    reviews.value = []
  } finally {
    reviewsLoading.value = false
  }
}

function handleSelectCycle(id: string) {
  selectedCycleId.value = id
  loadReviews(id)
}

async function handleCreateCycle() {
  try {
    const cycle = await createReviewCycle(cycleForm.value)
    cycles.value.unshift(cycle)
    showCycleModal.value = false
    message.success('Review cycle created')
  } catch {
    message.error('Failed to create cycle')
  }
}

async function handleAddReview() {
  if (!selectedCycleId.value) return
  try {
    await createReview(selectedCycleId.value, {
      employee_id: reviewForm.value.employee_id,
      reviewer_id: reviewForm.value.reviewer_id || undefined,
    })
    await loadReviews(selectedCycleId.value)
    showReviewModal.value = false
    message.success('Review created')
  } catch {
    message.error('Failed to create review')
  }
}

function openDetailModal(r: PerformanceReview) {
  editingReview.value = r
  detailForm.value = {
    status: r.status,
    self_rating: r.self_rating ?? undefined,
    manager_rating: r.manager_rating ?? undefined,
    self_summary: r.self_summary,
    manager_summary: r.manager_summary,
    strengths: r.strengths,
    improvements: r.improvements,
  }
  showDetailModal.value = true
}

async function handleSaveReview() {
  if (!selectedCycleId.value || !editingReview.value) return
  try {
    await updateReview(selectedCycleId.value, editingReview.value.id, detailForm.value)
    await loadReviews(selectedCycleId.value)
    showDetailModal.value = false
    message.success('Review updated')
  } catch {
    message.error('Failed to update review')
  }
}

const cycleStatusType = (s: string) => s === 'active' ? 'success' : s === 'completed' ? 'info' : 'default'
const reviewStatusType = (s: string) => {
  if (s === 'acknowledged') return 'success'
  if (s === 'submitted') return 'info'
  if (s === 'in_progress') return 'warning'
  return 'default'
}

const employeeOptions = computed(() =>
  employees.value.map((e) => ({ label: e.name, value: e.id }))
)

const reviewColumns: DataTableColumns<PerformanceReview> = [
  { title: 'Employee', key: 'employee_name' },
  {
    title: 'Status', key: 'status',
    render: (r) => h(NTag, { type: reviewStatusType(r.status), size: 'small' }, () => r.status),
  },
  { title: 'Self Rating', key: 'self_rating', render: (r) => r.self_rating ?? '—' },
  { title: 'Manager Rating', key: 'manager_rating', render: (r) => r.manager_rating ?? '—' },
  {
    title: 'Action', key: 'action',
    render: (r) => h(NButton, { size: 'small', onClick: () => openDetailModal(r) }, () => 'Edit'),
  },
]
</script>

<script lang="ts">
import { h } from 'vue'
export default {}
</script>

<template>
  <div>
    <PageHeader title="Performance Reviews">
      <template #actions>
        <NButton type="primary" @click="showCycleModal = true">
          <template #icon><NIcon :component="AddOutline" /></template>
          New Cycle
        </NButton>
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <div style="display: flex; gap: 16px; flex-wrap: wrap; margin-bottom: 20px">
        <NCard
          v-for="c in cycles" :key="c.id"
          size="small"
          :bordered="c.id === selectedCycleId"
          :style="{ cursor: 'pointer', minWidth: '180px', borderColor: c.id === selectedCycleId ? '#18a058' : undefined }"
          @click="handleSelectCycle(c.id)"
        >
          <div style="font-weight: 600">{{ c.title }}</div>
          <div style="font-size: 12px; color: #888">{{ c.period }}</div>
          <NTag :type="cycleStatusType(c.status)" size="small" style="margin-top: 4px">{{ c.status }}</NTag>
        </NCard>
      </div>

      <div v-if="selectedCycleId" style="margin-bottom: 12px">
        <NButton size="small" type="primary" @click="showReviewModal = true; reviewForm = { employee_id: '', reviewer_id: '' }">
          <template #icon><NIcon :component="AddOutline" /></template>
          Add Review
        </NButton>
      </div>

      <NSpin :show="reviewsLoading">
        <EmptyState v-if="reviews.length === 0 && !reviewsLoading" description="No reviews in this cycle" />
        <NDataTable v-else :columns="reviewColumns" :data="reviews" :bordered="false" size="small" />
      </NSpin>
    </NSpin>

    <!-- Create Cycle Modal -->
    <NModal v-model:show="showCycleModal" preset="card" title="New Review Cycle" style="max-width: 460px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Title" :show-feedback="false">
          <NInput v-model:value="cycleForm.title" placeholder="Q1 2026 Review" />
        </NFormItem>
        <NFormItem label="Period" :show-feedback="false">
          <NInput v-model:value="cycleForm.period" placeholder="2026-Q1" />
        </NFormItem>
        <NSpace :size="12">
          <NFormItem label="Start Date" :show-feedback="false">
            <input v-model="cycleForm.start_date" type="date" style="padding: 6px 10px; border: 1px solid #e0e0e6; border-radius: 3px; font-size: 14px" />
          </NFormItem>
          <NFormItem label="End Date" :show-feedback="false">
            <input v-model="cycleForm.end_date" type="date" style="padding: 6px 10px; border: 1px solid #e0e0e6; border-radius: 3px; font-size: 14px" />
          </NFormItem>
        </NSpace>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCycleModal = false">Cancel</NButton>
          <NButton type="primary" :disabled="!cycleForm.title || !cycleForm.period" @click="handleCreateCycle">Create</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Add Review Modal -->
    <NModal v-model:show="showReviewModal" preset="card" title="Add Review" style="max-width: 400px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Employee" :show-feedback="false">
          <NSelect v-model:value="reviewForm.employee_id" :options="employeeOptions" placeholder="Select employee" />
        </NFormItem>
        <NFormItem label="Reviewer (optional)" :show-feedback="false">
          <NSelect v-model:value="reviewForm.reviewer_id" :options="employeeOptions" placeholder="Select reviewer" clearable />
        </NFormItem>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showReviewModal = false">Cancel</NButton>
          <NButton type="primary" :disabled="!reviewForm.employee_id" @click="handleAddReview">Add</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Review Detail Modal -->
    <NModal v-model:show="showDetailModal" preset="card" :title="editingReview?.employee_name ?? 'Review'" style="max-width: 560px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Status" :show-feedback="false">
          <NSelect v-model:value="detailForm.status" :options="[
            { label: 'Pending', value: 'pending' },
            { label: 'In Progress', value: 'in_progress' },
            { label: 'Submitted', value: 'submitted' },
            { label: 'Acknowledged', value: 'acknowledged' },
          ]" />
        </NFormItem>
        <NSpace :size="16">
          <NFormItem label="Self Rating" :show-feedback="false">
            <NRate v-model:value="detailForm.self_rating" :count="5" />
          </NFormItem>
          <NFormItem label="Manager Rating" :show-feedback="false">
            <NRate v-model:value="detailForm.manager_rating" :count="5" />
          </NFormItem>
        </NSpace>
        <NFormItem label="Self Summary" :show-feedback="false">
          <NInput v-model:value="detailForm.self_summary" type="textarea" :rows="3" />
        </NFormItem>
        <NFormItem label="Manager Summary" :show-feedback="false">
          <NInput v-model:value="detailForm.manager_summary" type="textarea" :rows="3" />
        </NFormItem>
        <NFormItem label="Strengths" :show-feedback="false">
          <NInput v-model:value="detailForm.strengths" type="textarea" :rows="2" />
        </NFormItem>
        <NFormItem label="Areas for Improvement" :show-feedback="false">
          <NInput v-model:value="detailForm.improvements" type="textarea" :rows="2" />
        </NFormItem>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showDetailModal = false">Cancel</NButton>
          <NButton type="primary" @click="handleSaveReview">Save</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
