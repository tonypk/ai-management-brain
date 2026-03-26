<script setup lang="ts">
import { NCard, NTag } from 'naive-ui'
import type { Seat } from '@/types'

defineProps<{
  seat: Seat
  selected: boolean
}>()

const emit = defineEmits<{
  select: [seatType: string]
}>()

const seatEmojis: Record<string, string> = {
  ceo: '👔',
  cfo: '💰',
  cmo: '📢',
  cto: '💻',
  chro: '👥',
  coo: '⚙️',
}
</script>

<template>
  <NCard
    hoverable
    :bordered="true"
    :style="{
      borderColor: selected ? '#2080f0' : undefined,
      borderWidth: selected ? '2px' : undefined,
      cursor: 'pointer',
    }"
    content-style="padding: 16px"
    @click="emit('select', seat.seat_type)"
  >
    <div style="text-align: center">
      <div style="font-size: 28px; margin-bottom: 6px">
        {{ seatEmojis[seat.seat_type] || '🪑' }}
      </div>
      <div style="font-weight: 600; font-size: 14px; margin-bottom: 4px">
        {{ seat.title || seat.seat_type.toUpperCase() }}
      </div>
      <div style="color: #888; font-size: 12px; margin-bottom: 8px">
        {{ seat.scope }}
      </div>
      <NTag :type="seat.is_active ? 'success' : 'default'" size="small">
        {{ seat.is_active ? 'Active' : 'Inactive' }}
      </NTag>
    </div>
  </NCard>
</template>
