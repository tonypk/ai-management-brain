<script setup lang="ts">
import { ref } from 'vue'
import { NCard, NTag, NButton, NSpace, NIcon, NText } from 'naive-ui'
import { ChevronDownOutline, ChevronUpOutline, TrashOutline, EyeOutline } from '@vicons/ionicons5'
import type { BoardRecord } from '@/types'

defineProps<{
  record: BoardRecord
}>()

const emit = defineEmits<{
  view: [id: string]
  delete: [id: string]
}>()

const seatEmojis: Record<string, string> = {
  ceo: '👔', cfo: '💰', cmo: '📢', cto: '💻', chro: '👥', coo: '⚙️',
}

const expanded = ref(false)

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString()
}
</script>

<template>
  <NCard
    :bordered="false"
    size="small"
    style="box-shadow: 0 1px 3px rgba(0,0,0,0.08); margin-bottom: 12px"
  >
    <div style="display: flex; justify-content: space-between; align-items: flex-start">
      <div style="flex: 1; min-width: 0">
        <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px">
          <span style="font-size: 16px">📋</span>
          <NText strong style="font-size: 15px">{{ record.topic }}</NText>
        </div>
        <NText depth="3" style="font-size: 12px">{{ formatDate(record.created_at) }}</NText>
      </div>
      <NSpace :size="4">
        <NButton size="tiny" quaternary @click="expanded = !expanded">
          <template #icon>
            <NIcon :component="expanded ? ChevronUpOutline : ChevronDownOutline" />
          </template>
        </NButton>
        <NButton size="tiny" quaternary @click="emit('view', record.id)">
          <template #icon><NIcon :component="EyeOutline" /></template>
        </NButton>
        <NButton size="tiny" quaternary type="error" @click="emit('delete', record.id)">
          <template #icon><NIcon :component="TrashOutline" /></template>
        </NButton>
      </NSpace>
    </div>

    <NSpace :size="6" style="margin-top: 8px">
      <NTag
        v-for="resp in record.responses"
        :key="resp.seat_type"
        size="small"
        :bordered="false"
      >
        {{ seatEmojis[resp.seat_type] || '🪑' }} {{ resp.title || resp.seat_type.toUpperCase() }}
      </NTag>
    </NSpace>

    <div v-if="expanded" style="margin-top: 12px; padding-top: 12px; border-top: 1px solid #f0f0f0">
      <div style="font-weight: 600; margin-bottom: 6px; color: #16a34a; font-size: 13px">Synthesis</div>
      <NText style="white-space: pre-wrap; font-size: 13px; line-height: 1.6">{{ record.synthesis }}</NText>
    </div>
  </NCard>
</template>
