<script setup lang="ts">
import { NModal, NCard, NList, NListItem, NTag, NText, NSpace, NButton } from 'naive-ui'
import type { BoardRecord } from '@/types'

const props = defineProps<{
  record: BoardRecord | null
  show: boolean
}>()

const emit = defineEmits<{
  'update:show': [val: boolean]
}>()

const seatEmojis: Record<string, string> = {
  ceo: '👔', cfo: '💰', cmo: '📢', cto: '💻', chro: '👥', coo: '⚙️',
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString()
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    style="max-width: 700px; width: 95%"
    :title="record?.topic || 'Board Discussion'"
    :on-update:show="(val: boolean) => emit('update:show', val)"
  >
    <template v-if="record">
      <NText depth="3" style="font-size: 12px">{{ formatDate(record.created_at) }}</NText>

      <NList bordered style="margin-top: 16px">
        <NListItem v-for="resp in record.responses" :key="resp.seat_type">
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
        v-if="record.synthesis"
        style="margin-top: 16px; background: #f0fdf4; border: 1px solid #bbf7d0"
        :bordered="false"
        content-style="padding: 16px"
      >
        <div style="font-weight: 600; margin-bottom: 8px; color: #16a34a">Board Synthesis</div>
        <NText style="white-space: pre-wrap; font-size: 13px; line-height: 1.6">{{ record.synthesis }}</NText>
      </NCard>
    </template>

    <template #footer>
      <NSpace justify="end">
        <NButton @click="emit('update:show', false)">Close</NButton>
      </NSpace>
    </template>
  </NModal>
</template>
