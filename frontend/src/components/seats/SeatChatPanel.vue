<script setup lang="ts">
import { ref, computed } from 'vue'
import { NCard, NSelect, NInput, NButton, NSpace, NText, NEmpty, NSpin, NTag } from 'naive-ui'
import { chatWithSeat, type SeatChatResponse } from '@/api'
import type { Seat } from '@/types'

const props = defineProps<{
  seats: Seat[]
  selectedSeat: string
}>()

const emit = defineEmits<{
  'update:selectedSeat': [value: string]
}>()

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  seat_type?: string
}

const messageText = ref('')
const loading = ref(false)
const chatHistory = ref<ChatMessage[]>([])

const seatOptions = computed(() =>
  props.seats.map((s) => ({
    label: `${s.title || s.seat_type.toUpperCase()} - ${s.scope || ''}`,
    value: s.seat_type,
  })),
)

const seatEmojis: Record<string, string> = {
  ceo: '👔',
  cfo: '💰',
  cmo: '📢',
  cto: '💻',
  chro: '👥',
  coo: '⚙️',
}

async function handleSend() {
  if (!messageText.value.trim() || !props.selectedSeat) return
  const text = messageText.value.trim()
  chatHistory.value = [...chatHistory.value, { role: 'user', content: text }]
  messageText.value = ''
  loading.value = true
  try {
    const resp: SeatChatResponse = await chatWithSeat(props.selectedSeat, text)
    chatHistory.value = [...chatHistory.value, { role: 'assistant', content: resp.content, seat_type: resp.seat_type }]
  } catch (err: unknown) {
    chatHistory.value = [...chatHistory.value, { role: 'assistant', content: `Error: ${err instanceof Error ? err.message : 'Failed to get response'}` }]
  } finally {
    loading.value = false
  }
}

function handleSeatChange(val: string) {
  emit('update:selectedSeat', val)
  chatHistory.value = []
}
</script>

<template>
  <NCard :bordered="false">
    <NSpace vertical :size="16">
      <NSelect
        :value="selectedSeat"
        :options="seatOptions"
        placeholder="Select a C-Suite seat..."
        @update:value="handleSeatChange"
      />

      <div
        v-if="chatHistory.length > 0"
        style="max-height: 400px; overflow-y: auto; padding: 12px; background: #fafafa; border-radius: 8px"
      >
        <div v-for="(msg, i) in chatHistory" :key="i" :style="{ marginBottom: '12px' }">
          <div v-if="msg.role === 'user'" style="text-align: right">
            <NTag size="small" :bordered="false" style="margin-bottom: 4px">You</NTag>
            <div style="background: #e8f4fd; padding: 10px 14px; border-radius: 12px 12px 2px 12px; display: inline-block; max-width: 80%; text-align: left">
              <NText style="font-size: 13px">{{ msg.content }}</NText>
            </div>
          </div>
          <div v-else>
            <NSpace :size="4" align="center" style="margin-bottom: 4px">
              <span>{{ seatEmojis[msg.seat_type || selectedSeat] || '🪑' }}</span>
              <NTag size="small" type="info" :bordered="false">{{ (msg.seat_type || selectedSeat).toUpperCase() }}</NTag>
            </NSpace>
            <div style="background: #fff; padding: 10px 14px; border-radius: 2px 12px 12px 12px; border: 1px solid #eee; max-width: 80%">
              <NText style="white-space: pre-wrap; font-size: 13px; line-height: 1.6">{{ msg.content }}</NText>
            </div>
          </div>
        </div>
      </div>

      <NEmpty v-else-if="selectedSeat" description="Start a conversation with the selected C-Suite member" />
      <NEmpty v-else description="Select a seat to begin chatting" />

      <NSpace :size="12">
        <NInput
          v-model:value="messageText"
          placeholder="Type your message..."
          style="flex: 1; min-width: 300px"
          :disabled="loading || !selectedSeat"
          @keyup.enter="handleSend"
        />
        <NButton
          type="primary"
          :loading="loading"
          :disabled="!messageText.trim() || !selectedSeat"
          @click="handleSend"
        >
          Send
        </NButton>
      </NSpace>
      <NSpin v-if="loading" :show="true" size="small" style="display: flex; justify-content: center" />
    </NSpace>
  </NCard>
</template>
