<script setup lang="ts">
import { ref } from 'vue'
import { NModal, NCard, NInput, NButton, NSpace } from 'naive-ui'

defineProps<{ show: boolean }>()
const emit = defineEmits<{
  'update:show': [value: boolean]
  submit: [feedback: string]
}>()

const feedback = ref('')
const loading = ref(false)

function handleSubmit() {
  if (!feedback.value.trim()) return
  loading.value = true
  emit('submit', feedback.value.trim())
}

function handleClose() {
  feedback.value = ''
  loading.value = false
  emit('update:show', false)
}
</script>

<template>
  <NModal :show="show" @update:show="handleClose">
    <NCard
      title="Adjust Organization Plan"
      :bordered="false"
      style="width: 560px; max-width: 90vw"
      closable
      @close="handleClose"
    >
      <p style="color: #666; margin-bottom: 16px">
        Describe what you'd like to change. AI will regenerate the plan based on your feedback.
      </p>
      <NInput
        v-model:value="feedback"
        type="textarea"
        placeholder="e.g. Add a marketing department with 5 people, change management framework to OKR-based..."
        :rows="5"
        :disabled="loading"
      />
      <template #action>
        <NSpace justify="end">
          <NButton @click="handleClose" :disabled="loading">Cancel</NButton>
          <NButton
            type="primary"
            :loading="loading"
            :disabled="!feedback.trim()"
            @click="handleSubmit"
          >
            Adjust Plan
          </NButton>
        </NSpace>
      </template>
    </NCard>
  </NModal>
</template>
