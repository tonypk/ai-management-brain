<script setup lang="ts">
import { NDatePicker, NButton, NSpace, NIcon } from 'naive-ui'
import { ChevronBackOutline, ChevronForwardOutline } from '@vicons/ionicons5'

const props = defineProps<{
  modelValue: number
}>()

const emit = defineEmits<{
  'update:modelValue': [value: number]
}>()

function shiftDay(offset: number): void {
  const d = new Date(props.modelValue)
  d.setDate(d.getDate() + offset)
  emit('update:modelValue', d.getTime())
}
</script>

<template>
  <NSpace align="center" :size="8">
    <NButton quaternary size="small" @click="shiftDay(-1)">
      <template #icon><NIcon :component="ChevronBackOutline" /></template>
    </NButton>
    <NDatePicker
      :value="modelValue"
      type="date"
      :clearable="false"
      style="width: 160px"
      @update:value="emit('update:modelValue', $event)"
    />
    <NButton quaternary size="small" @click="shiftDay(1)">
      <template #icon><NIcon :component="ChevronForwardOutline" /></template>
    </NButton>
  </NSpace>
</template>
