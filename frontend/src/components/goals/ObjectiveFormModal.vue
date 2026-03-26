<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { NModal, NInput, NSelect, NButton, NSpace, NFormItem } from 'naive-ui'
import GoalCycleSelector from './GoalCycleSelector.vue'
import type { Objective, GoalStatus } from '@/types'
import { listEmployees } from '@/api/employees'

const props = defineProps<{
  show: boolean
  objective?: Objective | null
  defaultCycle: string
}>()

const emit = defineEmits<{
  'update:show': [val: boolean]
  submit: [data: { title: string; description: string; status: GoalStatus; cycle: string; owner_id: string | null }]
}>()

const title = ref('')
const description = ref('')
const status = ref<GoalStatus>('draft')
const cycle = ref('')
const ownerId = ref<string | null>(null)
const ownerOptions = ref<{ label: string; value: string }[]>([])

const statusOptions = [
  { label: 'Draft', value: 'draft' },
  { label: 'Active', value: 'active' },
  { label: 'Completed', value: 'completed' },
  { label: 'Cancelled', value: 'cancelled' },
]

onMounted(async () => {
  try {
    const employees = await listEmployees()
    ownerOptions.value = employees.map((e) => ({ label: e.name, value: e.id }))
  } catch { /* ignore */ }
})

watch(() => props.show, (val) => {
  if (val) {
    if (props.objective) {
      title.value = props.objective.title
      description.value = props.objective.description
      status.value = props.objective.status
      cycle.value = props.objective.cycle
      ownerId.value = props.objective.owner_id ?? null
    } else {
      title.value = ''
      description.value = ''
      status.value = 'draft'
      cycle.value = props.defaultCycle
      ownerId.value = null
    }
  }
})

function handleSubmit() {
  if (!title.value.trim()) return
  emit('submit', {
    title: title.value.trim(),
    description: description.value.trim(),
    status: status.value,
    cycle: cycle.value,
    owner_id: ownerId.value,
  })
  emit('update:show', false)
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    style="max-width: 500px; width: 95%"
    :title="objective ? 'Edit Objective/OKR' : 'New Objective/OKR'"
    :on-update:show="(val: boolean) => emit('update:show', val)"
  >
    <NSpace vertical :size="12">
      <NFormItem label="Title" :show-feedback="false">
        <NInput v-model:value="title" placeholder="Objective title" />
      </NFormItem>
      <NFormItem label="Description" :show-feedback="false">
        <NInput v-model:value="description" type="textarea" :rows="2" placeholder="Optional description" />
      </NFormItem>
      <NFormItem label="Owner" :show-feedback="false">
        <NSelect v-model:value="ownerId" :options="ownerOptions" clearable placeholder="Select owner" />
      </NFormItem>
      <NSpace :size="12">
        <NFormItem label="Status" :show-feedback="false">
          <NSelect v-model:value="status" :options="statusOptions" style="width: 140px" />
        </NFormItem>
        <NFormItem label="Cycle" :show-feedback="false">
          <GoalCycleSelector v-model="cycle" />
        </NFormItem>
      </NSpace>
    </NSpace>

    <template #footer>
      <NSpace justify="end">
        <NButton @click="emit('update:show', false)">Cancel</NButton>
        <NButton type="primary" :disabled="!title.trim()" @click="handleSubmit">
          {{ objective ? 'Save' : 'Create' }}
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>
