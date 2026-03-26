<script setup lang="ts">
import { ref } from 'vue'
import { NCard, NButton, NIcon, NSpace, NText, NProgress, useDialog, useMessage } from 'naive-ui'
import { AddOutline, CreateOutline, TrashOutline } from '@vicons/ionicons5'
import GoalStatusBadge from './GoalStatusBadge.vue'
import KeyResultList from './KeyResultList.vue'
import KeyResultFormModal from './KeyResultFormModal.vue'
import { usePlanningStore } from '@/stores/planning'
import type { Objective, KeyResult } from '@/types'

const props = defineProps<{
  objective: Objective
  employeeMap?: Record<string, string>
}>()

const emit = defineEmits<{
  edit: [obj: Objective]
}>()

const store = usePlanningStore()
const dialog = useDialog()
const message = useMessage()

const showKrModal = ref(false)
const editingKr = ref<KeyResult | null>(null)

function overallPercent(): number {
  const krs = props.objective.key_results
  if (krs.length === 0) return 0
  const sum = krs.reduce((acc, kr) => {
    return acc + (kr.target > 0 ? Math.min((kr.current_value / kr.target) * 100, 100) : 0)
  }, 0)
  return Math.round(sum / krs.length)
}

function handleAddKr() {
  editingKr.value = null
  showKrModal.value = true
}

function handleEditKr(kr: KeyResult) {
  editingKr.value = kr
  showKrModal.value = true
}

function handleKrSubmit(data: { title: string; target: number; current_value: number; unit: string; due_date: string }) {
  if (editingKr.value) {
    store.updateKeyResult(props.objective.id, editingKr.value.id, data)
    message.success('Key result updated')
  } else {
    store.addKeyResult(props.objective.id, data.title, data.target, data.unit, data.due_date)
    message.success('Key result added')
  }
}

function handleDeleteKr(krId: string) {
  dialog.warning({
    title: 'Delete Key Result',
    content: 'Are you sure?',
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: () => store.deleteKeyResult(props.objective.id, krId),
  })
}

function handleDeleteObjective() {
  dialog.warning({
    title: 'Delete Objective',
    content: `Delete "${props.objective.title}" and all its key results?`,
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: () => store.deleteObjective(props.objective.id),
  })
}
</script>

<template>
  <NCard :bordered="false" size="small" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 8px">
      <div style="flex: 1; min-width: 0">
        <NText strong style="font-size: 15px">{{ objective.title }}</NText>
        <div style="margin-top: 4px">
          <GoalStatusBadge :status="objective.status" />
        </div>
      </div>
      <NSpace :size="4">
        <NButton size="tiny" quaternary @click="emit('edit', objective)">
          <template #icon><NIcon :component="CreateOutline" /></template>
        </NButton>
        <NButton size="tiny" quaternary type="error" @click="handleDeleteObjective">
          <template #icon><NIcon :component="TrashOutline" /></template>
        </NButton>
      </NSpace>
    </div>

    <NText v-if="objective.description" depth="3" style="font-size: 12px; display: block; margin-bottom: 8px">
      {{ objective.description }}
    </NText>

    <NProgress
      type="line"
      :percentage="overallPercent()"
      :show-indicator="true"
      style="margin-bottom: 8px"
    />

    <div style="font-size: 12px; color: #888; margin-bottom: 8px">
      Key Results: {{ objective.key_results.length }}
      <span v-if="objective.owner_id && employeeMap?.[objective.owner_id]"> · Owner: {{ employeeMap[objective.owner_id] }}</span>
    </div>

    <KeyResultList
      :key-results="objective.key_results"
      :objective-id="objective.id"
      @edit="handleEditKr"
      @delete="handleDeleteKr"
    />

    <NButton
      size="small"
      dashed
      style="margin-top: 8px; width: 100%"
      @click="handleAddKr"
    >
      <template #icon><NIcon :component="AddOutline" /></template>
      Add Key Result
    </NButton>

    <KeyResultFormModal
      v-model:show="showKrModal"
      :key-result="editingKr"
      @submit="handleKrSubmit"
    />
  </NCard>
</template>
