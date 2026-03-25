<script setup lang="ts">
import { ref, watch } from 'vue'
import { NModal, NInput, NSelect, NButton, NSpace, NFormItem } from 'naive-ui'
import GoalCycleSelector from './GoalCycleSelector.vue'
import type { Objective, GoalStatus } from '@/types'

const props = defineProps<{
  show: boolean
  objective?: Objective | null
  defaultCycle: string
}>()

const emit = defineEmits<{
  'update:show': [val: boolean]
  submit: [data: { title: string; description: string; status: GoalStatus; cycle: string }]
}>()

const title = ref('')
const description = ref('')
const status = ref<GoalStatus>('draft')
const cycle = ref('')

const statusOptions = [
  { label: 'Draft', value: 'draft' },
  { label: 'Active', value: 'active' },
  { label: 'Completed', value: 'completed' },
  { label: 'Cancelled', value: 'cancelled' },
]

watch(() => props.show, (val) => {
  if (val) {
    if (props.objective) {
      title.value = props.objective.title
      description.value = props.objective.description
      status.value = props.objective.status
      cycle.value = props.objective.cycle
    } else {
      title.value = ''
      description.value = ''
      status.value = 'draft'
      cycle.value = props.defaultCycle
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
  })
  emit('update:show', false)
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    style="max-width: 500px; width: 95%"
    :title="objective ? 'Edit Objective' : 'New Objective'"
    :on-update:show="(val: boolean) => emit('update:show', val)"
  >
    <NSpace vertical :size="12">
      <NFormItem label="Title" :show-feedback="false">
        <NInput v-model:value="title" placeholder="Objective title" />
      </NFormItem>
      <NFormItem label="Description" :show-feedback="false">
        <NInput v-model:value="description" type="textarea" :rows="2" placeholder="Optional description" />
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
