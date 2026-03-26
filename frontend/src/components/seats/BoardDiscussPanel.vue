<script setup lang="ts">
import { ref } from 'vue'
import { NCard, NInput, NButton, NSpace, NList, NListItem, NTag, NText, NSpin, NEmpty } from 'naive-ui'
import { boardDiscuss } from '@/api'
import type { BoardDiscussResult } from '@/types'

const topic = ref('')
const loading = ref(false)
const result = ref<BoardDiscussResult | null>(null)
const error = ref('')

const seatEmojis: Record<string, string> = {
  ceo: '👔',
  cfo: '💰',
  cmo: '📢',
  cto: '💻',
  chro: '👥',
  coo: '⚙️',
}

async function handleDiscuss() {
  if (!topic.value.trim()) return
  loading.value = true
  error.value = ''
  result.value = null
  try {
    result.value = await boardDiscuss(topic.value.trim())
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : 'Discussion failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <NCard :bordered="false">
    <NSpace vertical :size="16">
      <NSpace :size="12">
        <NInput
          v-model:value="topic"
          placeholder="Enter a topic for the board to discuss..."
          style="flex: 1; min-width: 300px"
          :disabled="loading"
          @keyup.enter="handleDiscuss"
        />
        <NButton type="primary" :loading="loading" :disabled="!topic.trim()" @click="handleDiscuss">
          Discuss
        </NButton>
      </NSpace>

      <NSpin :show="loading">
        <div v-if="error" style="color: #ef4444; padding: 12px">{{ error }}</div>

        <template v-if="result">
          <NList bordered>
            <NListItem v-for="resp in result.responses" :key="resp.seat_type">
              <template #prefix>
                <span style="font-size: 20px">{{ seatEmojis[resp.seat_type] || '🪑' }}</span>
              </template>
              <div>
                <NSpace align="center" :size="8" style="margin-bottom: 4px">
                  <NTag size="small" type="info">{{ resp.title || resp.seat_type.toUpperCase() }}</NTag>
                </NSpace>
                <NText style="white-space: pre-wrap; font-size: 13px; line-height: 1.6">{{ resp.content }}</NText>
              </div>
            </NListItem>
          </NList>

          <NCard
            v-if="result.synthesis"
            style="margin-top: 16px; background: #f0fdf4; border: 1px solid #bbf7d0"
            :bordered="false"
            content-style="padding: 16px"
          >
            <div style="font-weight: 600; margin-bottom: 8px; color: #16a34a">Board Synthesis</div>
            <NText style="white-space: pre-wrap; font-size: 13px; line-height: 1.6">{{ result.synthesis }}</NText>
          </NCard>
        </template>

        <NEmpty v-if="!result && !loading && !error" description="Enter a topic to start a board discussion" />
      </NSpin>
    </NSpace>
  </NCard>
</template>
