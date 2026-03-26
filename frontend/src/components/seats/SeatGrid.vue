<script setup lang="ts">
import { NGrid, NGridItem } from 'naive-ui'
import SeatCard from './SeatCard.vue'
import type { Seat } from '@/types'

defineProps<{
  seats: Seat[]
  selectedSeat: string
}>()

const emit = defineEmits<{
  select: [seatType: string]
}>()
</script>

<template>
  <NGrid :cols="24" :x-gap="16" :y-gap="16" responsive="screen">
    <NGridItem
      v-for="seat in seats"
      :key="seat.id"
      :span="24"
      :m="12"
      :l="8"
    >
      <SeatCard
        :seat="seat"
        :selected="seat.seat_type === selectedSeat"
        @select="emit('select', $event)"
      />
    </NGridItem>
  </NGrid>
</template>
