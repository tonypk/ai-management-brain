<script setup lang="ts">
import { ref } from 'vue'
import { NModal, NInput, NButton, NSpace, NSpin, useMessage } from 'naive-ui'
import { boardDiscuss } from '@/api'
import { usePlanningStore } from '@/stores/planning'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  'update:show': [val: boolean]
  created: []
}>()

const message = useMessage()
const store = usePlanningStore()

const topic = ref('')
const loading = ref(false)

async function handleSubmit() {
  const t = topic.value.trim()
  if (!t) return

  loading.value = true
  try {
    const result = await boardDiscuss(t)
    store.addBoardRecord(result.topic, result.responses, result.synthesis)
    message.success('Discussion saved')
    topic.value = ''
    emit('update:show', false)
    emit('created')
  } catch (err: unknown) {
    message.error(`Discussion failed: ${err instanceof Error ? err.message : 'Unknown error'}`)
  } finally {
    loading.value = false
  }
}

function handleClose(val: boolean) {
  if (!loading.value) {
    emit('update:show', val)
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    style="max-width: 500px; width: 95%"
    title="New Board Discussion"
    :mask-closable="!loading"
    :closable="!loading"
    :on-update:show="handleClose"
  >
    <NSpin :show="loading">
      <NSpace vertical :size="16">
        <NInput
          v-model:value="topic"
          type="textarea"
          placeholder="Enter a topic for the board to discuss..."
          :rows="3"
          :disabled="loading"
          @keyup.ctrl.enter="handleSubmit"
        />
        <div style="color: #888; font-size: 12px">
          The AI board will discuss this topic from all C-Suite perspectives. Press Ctrl+Enter to submit.
        </div>
      </NSpace>
    </NSpin>

    <template #footer>
      <NSpace justify="end">
        <NButton :disabled="loading" @click="handleClose(false)">Cancel</NButton>
        <NButton type="primary" :loading="loading" :disabled="!topic.trim()" @click="handleSubmit">
          Start Discussion
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>
