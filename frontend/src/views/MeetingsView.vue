<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  NButton, NIcon, NCard, NSpin, NDataTable, NTag, NText,
  NModal, NInput, NSelect, NSpace, NFormItem, NInputNumber,
  useMessage,
} from 'naive-ui'
import { AddOutline, CheckmarkOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import {
  listMeetings, createMeeting, updateMeeting,
  listActionItems, createActionItem, updateActionItem,
  listOpenActionItems,
} from '@/api/meetings'
import { listEmployees } from '@/api/employees'
import type { Meeting, ActionItem, OpenActionItem, Employee } from '@/types'

const message = useMessage()

const loading = ref(true)
const meetings = ref<Meeting[]>([])
const employees = ref<Employee[]>([])
const openActions = ref<OpenActionItem[]>([])


// Meeting modal
const showMeetingModal = ref(false)
const meetingForm = ref({
  employee_id: '',
  manager_id: '',
  meeting_date: new Date().toISOString().slice(0, 10),
  duration_min: 30,
  notes: '',
  mood: '',
  follow_up: '',
})

// Meeting detail modal
const showDetailModal = ref(false)
const editingMeeting = ref<Meeting | null>(null)
const detailForm = ref({ notes: '', mood: '', follow_up: '', duration_min: 30 })
const actionItems = ref<ActionItem[]>([])
const newActionTitle = ref('')

onMounted(async () => {
  try {
    const [m, e, a] = await Promise.all([listMeetings(), listEmployees(), listOpenActionItems()])
    meetings.value = m
    employees.value = e
    openActions.value = a
  } catch {
    message.error('Failed to load data')
  } finally {
    loading.value = false
  }
})

const employeeOptions = () =>
  employees.value.map((e) => ({ label: e.name, value: e.id }))

const moodOptions = [
  { label: 'Great', value: 'great' },
  { label: 'Good', value: 'good' },
  { label: 'Neutral', value: 'neutral' },
  { label: 'Concerning', value: 'concerning' },
  { label: 'Critical', value: 'critical' },
]

const moodColor = (m: string) => {
  if (m === 'great') return 'success'
  if (m === 'good') return 'info'
  if (m === 'concerning') return 'warning'
  if (m === 'critical') return 'error'
  return 'default'
}

async function handleCreateMeeting() {
  try {
    const m = await createMeeting({
      ...meetingForm.value,
      manager_id: meetingForm.value.manager_id || undefined,
    })
    meetings.value.unshift({ ...m, employee_name: employees.value.find((e) => e.id === m.employee_id)?.name ?? '' })
    showMeetingModal.value = false
    message.success('Meeting created')
  } catch {
    message.error('Failed to create meeting')
  }
}

async function openDetail(m: Meeting) {
  editingMeeting.value = m
  detailForm.value = { notes: m.notes, mood: m.mood, follow_up: m.follow_up, duration_min: m.duration_min }
  try {
    actionItems.value = await listActionItems(m.id)
  } catch {
    actionItems.value = []
  }
  showDetailModal.value = true
}

async function handleSaveMeeting() {
  if (!editingMeeting.value) return
  try {
    await updateMeeting(editingMeeting.value.id, detailForm.value)
    const idx = meetings.value.findIndex((m) => m.id === editingMeeting.value!.id)
    if (idx >= 0) Object.assign(meetings.value[idx], detailForm.value)
    message.success('Meeting updated')
  } catch {
    message.error('Failed to update')
  }
}

async function handleAddAction() {
  if (!editingMeeting.value || !newActionTitle.value.trim()) return
  try {
    const item = await createActionItem(editingMeeting.value.id, { title: newActionTitle.value.trim() })
    actionItems.value.push(item)
    newActionTitle.value = ''
  } catch {
    message.error('Failed to add action item')
  }
}

async function handleToggleAction(item: ActionItem) {
  if (!editingMeeting.value) return
  const newStatus = item.status === 'done' ? 'open' : 'done'
  try {
    await updateActionItem(editingMeeting.value.id, item.id, { ...item, status: newStatus })
    item.status = newStatus as ActionItem['status']
  } catch {
    message.error('Failed to update action')
  }
}

const meetingColumns: DataTableColumns<Meeting> = [
  { title: 'Date', key: 'meeting_date', width: 110 },
  { title: 'Employee', key: 'employee_name' },
  { title: 'Duration', key: 'duration_min', width: 80, render: (m) => `${m.duration_min}min` },
  {
    title: 'Mood', key: 'mood', width: 100,
    render: (m) => m.mood ? h(NTag, { type: moodColor(m.mood), size: 'small' }, () => m.mood) : '—',
  },
  {
    title: '', key: 'action', width: 80,
    render: (m) => h(NButton, { size: 'small', onClick: () => openDetail(m) }, () => 'View'),
  },
]
</script>

<script lang="ts">
import { h } from 'vue'
export default {}
</script>

<template>
  <div>
    <PageHeader title="1:1 Meetings">
      <template #actions>
        <NButton type="primary" @click="showMeetingModal = true">
          <template #icon><NIcon :component="AddOutline" /></template>
          New Meeting
        </NButton>
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <!-- Open Action Items summary -->
      <NCard v-if="openActions.length > 0" size="small" :bordered="false" style="margin-bottom: 16px">
        <div style="font-weight: 600; font-size: 14px; margin-bottom: 8px">Open Action Items ({{ openActions.length }})</div>
        <div v-for="a in openActions.slice(0, 5)" :key="a.id" style="padding: 4px 0; font-size: 13px; border-bottom: 1px solid #f5f5f5">
          <NText>{{ a.title }}</NText>
          <NText depth="3" style="margin-left: 8px">— {{ a.employee_name }}</NText>
          <NTag v-if="a.due_date" size="tiny" style="margin-left: 8px">{{ a.due_date }}</NTag>
        </div>
      </NCard>

      <EmptyState v-if="meetings.length === 0 && !loading" description="No meetings recorded yet" />
      <NDataTable v-else :columns="meetingColumns" :data="meetings" :bordered="false" size="small" />
    </NSpin>

    <!-- Create Meeting Modal -->
    <NModal v-model:show="showMeetingModal" preset="card" title="New 1:1 Meeting" style="max-width: 460px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Employee" :show-feedback="false">
          <NSelect v-model:value="meetingForm.employee_id" :options="employeeOptions()" placeholder="Select employee" />
        </NFormItem>
        <NFormItem label="Date" :show-feedback="false">
          <input v-model="meetingForm.meeting_date" type="date" style="padding: 6px 10px; border: 1px solid #e0e0e6; border-radius: 3px; font-size: 14px" />
        </NFormItem>
        <NSpace :size="12">
          <NFormItem label="Duration (min)" :show-feedback="false">
            <NInputNumber v-model:value="meetingForm.duration_min" :min="5" :max="180" style="width: 110px" />
          </NFormItem>
          <NFormItem label="Mood" :show-feedback="false">
            <NSelect v-model:value="meetingForm.mood" :options="moodOptions" clearable style="width: 140px" />
          </NFormItem>
        </NSpace>
        <NFormItem label="Notes" :show-feedback="false">
          <NInput v-model:value="meetingForm.notes" type="textarea" :rows="3" />
        </NFormItem>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showMeetingModal = false">Cancel</NButton>
          <NButton type="primary" :disabled="!meetingForm.employee_id" @click="handleCreateMeeting">Create</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Meeting Detail Modal -->
    <NModal v-model:show="showDetailModal" preset="card" :title="editingMeeting?.employee_name + ' — ' + editingMeeting?.meeting_date" style="max-width: 560px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Notes" :show-feedback="false">
          <NInput v-model:value="detailForm.notes" type="textarea" :rows="4" />
        </NFormItem>
        <NSpace :size="12">
          <NFormItem label="Mood" :show-feedback="false">
            <NSelect v-model:value="detailForm.mood" :options="moodOptions" clearable style="width: 140px" />
          </NFormItem>
          <NFormItem label="Duration (min)" :show-feedback="false">
            <NInputNumber v-model:value="detailForm.duration_min" :min="5" :max="180" style="width: 110px" />
          </NFormItem>
        </NSpace>
        <NFormItem label="Follow Up" :show-feedback="false">
          <NInput v-model:value="detailForm.follow_up" type="textarea" :rows="2" />
        </NFormItem>

        <div style="font-weight: 600; font-size: 14px; margin-top: 8px">Action Items</div>
        <div v-for="item in actionItems" :key="item.id" style="display: flex; align-items: center; gap: 8px; padding: 4px 0">
          <NButton size="tiny" :type="item.status === 'done' ? 'success' : 'default'" @click="handleToggleAction(item)">
            <template #icon><NIcon :component="CheckmarkOutline" /></template>
          </NButton>
          <NText :style="{ textDecoration: item.status === 'done' ? 'line-through' : 'none', fontSize: '13px' }">{{ item.title }}</NText>
        </div>
        <div style="display: flex; gap: 8px">
          <NInput v-model:value="newActionTitle" placeholder="New action item..." size="small" @keyup.enter="handleAddAction" />
          <NButton size="small" @click="handleAddAction" :disabled="!newActionTitle.trim()">Add</NButton>
        </div>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showDetailModal = false">Close</NButton>
          <NButton type="primary" @click="handleSaveMeeting">Save</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
