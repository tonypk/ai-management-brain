<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NSpin, NSpace, NTabs, NTabPane, useMessage } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import SeatGrid from '@/components/seats/SeatGrid.vue'
import BoardDiscussPanel from '@/components/seats/BoardDiscussPanel.vue'
import SeatChatPanel from '@/components/seats/SeatChatPanel.vue'
import { listSeats } from '@/api'
import type { Seat } from '@/types'

const message = useMessage()

const loading = ref(true)
const seats = ref<Seat[]>([])
const selectedSeat = ref('')

async function fetchSeats() {
  loading.value = true
  try {
    seats.value = await listSeats()
    if (seats.value.length > 0 && !selectedSeat.value) {
      selectedSeat.value = seats.value[0].seat_type
    }
  } catch (err: unknown) {
    message.error(`Failed to load seats: ${err instanceof Error ? err.message : 'Unknown error'}`)
  } finally {
    loading.value = false
  }
}

function handleSeatSelect(seatType: string) {
  selectedSeat.value = seatType
}

onMounted(fetchSeats)
</script>

<template>
  <div>
    <PageHeader title="AI C-Suite Board" />

    <NSpin :show="loading">
      <NSpace vertical :size="24">
        <SeatGrid
          :seats="seats"
          :selected-seat="selectedSeat"
          @select="handleSeatSelect"
        />

        <NTabs type="line" animated>
          <NTabPane name="board" tab="Board Discussion">
            <BoardDiscussPanel />
          </NTabPane>
          <NTabPane name="chat" tab="Chat with Seat">
            <SeatChatPanel
              :seats="seats"
              :selected-seat="selectedSeat"
              @update:selected-seat="selectedSeat = $event"
            />
          </NTabPane>
        </NTabs>
      </NSpace>
    </NSpin>
  </div>
</template>
