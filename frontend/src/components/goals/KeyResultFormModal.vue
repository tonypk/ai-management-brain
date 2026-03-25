<script setup lang="ts">
import { ref, watch } from 'vue'
import { NModal, NInput, NInputNumber, NButton, NSpace, NFormItem } from 'naive-ui'
import type { KeyResult } from '@/types'

const props = defineProps<{
  show: boolean
  keyResult?: KeyResult | null
}>()

const emit = defineEmits<{
  'update:show': [val: boolean]
  submit: [data: { title: string; target: number; current_value: number; unit: string; due_date: string }]
}>()

const title = ref('')
const target = ref<number>(100)
const currentValue = ref<number>(0)
const unit = ref('%')
const dueDate = ref('')

watch(() => props.show, (val) => {
  if (val) {
    if (props.keyResult) {
      title.value = props.keyResult.title
      target.value = props.keyResult.target
      currentValue.value = props.keyResult.current_value
      unit.value = props.keyResult.unit
      dueDate.value = props.keyResult.due_date ?? ''
    } else {
      title.value = ''
      target.value = 100
      currentValue.value = 0
      unit.value = '%'
      // Default due date: end of current quarter
      const now = new Date()
      const q = Math.ceil((now.getMonth() + 1) / 3)
      const endMonth = q * 3
      const lastDay = new Date(now.getFullYear(), endMonth, 0)
      dueDate.value = lastDay.toISOString().slice(0, 10)
    }
  }
})

function handleSubmit() {
  if (!title.value.trim()) return
  emit('submit', {
    title: title.value.trim(),
    target: target.value,
    current_value: currentValue.value,
    unit: unit.value,
    due_date: dueDate.value,
  })
  emit('update:show', false)
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    style="max-width: 460px; width: 95%"
    :title="keyResult ? 'Edit Key Result' : 'New Key Result'"
    :on-update:show="(val: boolean) => emit('update:show', val)"
  >
    <NSpace vertical :size="12">
      <NFormItem label="Key Result" :show-feedback="false">
        <NInput v-model:value="title" placeholder="e.g. Reduce churn to <5%" />
      </NFormItem>
      <NSpace :size="12">
        <NFormItem label="Target" :show-feedback="false">
          <NInputNumber v-model:value="target" :min="0" style="width: 110px" />
        </NFormItem>
        <NFormItem label="Current" :show-feedback="false">
          <NInputNumber v-model:value="currentValue" :min="0" style="width: 110px" />
        </NFormItem>
        <NFormItem label="Unit" :show-feedback="false">
          <NInput v-model:value="unit" placeholder="%" style="width: 80px" />
        </NFormItem>
      </NSpace>
      <NFormItem label="Due Date" :show-feedback="false">
        <input
          v-model="dueDate"
          type="date"
          style="width: 160px; padding: 6px 10px; border: 1px solid #e0e0e6; border-radius: 3px; font-size: 14px"
        />
      </NFormItem>
    </NSpace>

    <template #footer>
      <NSpace justify="end">
        <NButton @click="emit('update:show', false)">Cancel</NButton>
        <NButton type="primary" :disabled="!title.trim()" @click="handleSubmit">
          {{ keyResult ? 'Save' : 'Add' }}
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>
