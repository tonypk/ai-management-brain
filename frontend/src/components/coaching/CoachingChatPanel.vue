<script setup lang="ts">
import { ref, nextTick, watch } from 'vue'
import { NCard, NInput, NButton, NSelect, NSpace, NSpin } from 'naive-ui'
import { chatWithSeat } from '@/api/seats'
import type { AtRiskEmployee, CoachingMessage } from '@/types'

const props = defineProps<{
  employees: AtRiskEmployee[]
  selectedId?: string
}>()

const selectedEmployee = ref<string | null>(props.selectedId ?? null)
const inputText = ref('')
const messages = ref<CoachingMessage[]>([])
const sending = ref(false)
const chatContainer = ref<HTMLElement>()

watch(() => props.selectedId, (id) => {
  if (id) selectedEmployee.value = id
})

const employeeOptions = props.employees.map(e => ({
  label: e.name,
  value: e.id,
}))

function getEmployeeContext(): string {
  const emp = props.employees.find(e => e.id === selectedEmployee.value)
  if (!emp) return ''
  return `Employee: ${emp.name}, Risk: ${emp.risk}, Missed ${emp.missed_7d} days in 7d, Sentiment: ${emp.last_sentiment}`
}

async function sendMessage(): Promise<void> {
  const text = inputText.value.trim()
  if (!text || sending.value) return

  const context = getEmployeeContext()
  const fullMessage = context ? `[Context: ${context}]\n\n${text}` : text

  messages.value = [...messages.value, { role: 'user', content: text }]
  inputText.value = ''
  sending.value = true

  await nextTick()
  scrollToBottom()

  try {
    const response = await chatWithSeat('chro', fullMessage)
    messages.value = [...messages.value, { role: 'assistant', content: response.content }]
  } catch {
    messages.value = [...messages.value, { role: 'assistant', content: 'Failed to get response. Please try again.' }]
  } finally {
    sending.value = false
    await nextTick()
    scrollToBottom()
  }
}

function scrollToBottom(): void {
  if (chatContainer.value) {
    chatContainer.value.scrollTop = chatContainer.value.scrollHeight
  }
}

function handleKeydown(e: KeyboardEvent): void {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}
</script>

<template>
  <NCard title="CHRO Coach" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08); display: flex; flex-direction: column">
    <!-- Employee selector -->
    <NSelect
      v-model:value="selectedEmployee"
      :options="employeeOptions"
      placeholder="Select employee..."
      clearable
      size="small"
      style="margin-bottom: 12px"
    />

    <!-- Chat messages -->
    <div
      ref="chatContainer"
      style="flex: 1; min-height: 300px; max-height: 500px; overflow-y: auto; padding: 8px 0; border-top: 1px solid #f0f0f0; border-bottom: 1px solid #f0f0f0"
    >
      <div v-if="!messages.length" style="color: #888; font-size: 13px; text-align: center; padding: 40px 16px">
        Ask the CHRO AI coach for advice on how to approach 1:1 conversations with your team members.
      </div>
      <div v-for="(msg, idx) in messages" :key="idx" style="margin-bottom: 12px; padding: 0 4px">
        <div :style="{ textAlign: msg.role === 'user' ? 'right' : 'left' }">
          <span style="font-size: 11px; color: #888; margin-bottom: 2px; display: block">
            {{ msg.role === 'user' ? 'You' : 'CHRO Coach' }}
          </span>
          <div
            :style="{
              display: 'inline-block',
              maxWidth: '85%',
              padding: '8px 12px',
              borderRadius: '12px',
              fontSize: '13px',
              lineHeight: '1.5',
              textAlign: 'left',
              background: msg.role === 'user' ? '#6366f1' : '#f5f5f5',
              color: msg.role === 'user' ? '#fff' : '#333',
              whiteSpace: 'pre-wrap',
            }"
          >
            {{ msg.content }}
          </div>
        </div>
      </div>
      <div v-if="sending" style="padding: 0 4px">
        <span style="font-size: 11px; color: #888">CHRO Coach</span>
        <NSpin size="small" style="margin-left: 8px" />
      </div>
    </div>

    <!-- Input -->
    <NSpace style="margin-top: 12px" :wrap="false">
      <NInput
        v-model:value="inputText"
        type="textarea"
        placeholder="Ask the CHRO coach..."
        :autosize="{ minRows: 1, maxRows: 3 }"
        :disabled="sending"
        style="flex: 1"
        @keydown="handleKeydown"
      />
      <NButton
        type="primary"
        :loading="sending"
        :disabled="!inputText.trim()"
        @click="sendMessage"
      >
        Send
      </NButton>
    </NSpace>
  </NCard>
</template>
