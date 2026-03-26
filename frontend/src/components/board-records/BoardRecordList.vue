<script setup lang="ts">
import EmptyState from '@/components/shared/EmptyState.vue'
import BoardRecordCard from './BoardRecordCard.vue'
import type { BoardRecord } from '@/types'

defineProps<{
  records: BoardRecord[]
}>()

defineEmits<{
  view: [id: string]
  delete: [id: string]
}>()
</script>

<template>
  <div>
    <EmptyState v-if="records.length === 0" description="No board discussions yet" />
    <BoardRecordCard
      v-for="record in records"
      :key="record.id"
      :record="record"
      @view="$emit('view', $event)"
      @delete="$emit('delete', $event)"
    />
  </div>
</template>
