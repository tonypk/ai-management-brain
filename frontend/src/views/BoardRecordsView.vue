<script setup lang="ts">
import { ref, computed } from 'vue'
import { NButton, NInput, NIcon, useDialog } from 'naive-ui'
import { AddOutline, SearchOutline } from '@vicons/ionicons5'
import PageHeader from '@/components/shared/PageHeader.vue'
import BoardRecordList from '@/components/board-records/BoardRecordList.vue'
import BoardRecordDetail from '@/components/board-records/BoardRecordDetail.vue'
import NewDiscussionModal from '@/components/board-records/NewDiscussionModal.vue'
import { usePlanningStore } from '@/stores/planning'
import type { BoardRecord } from '@/types'

const store = usePlanningStore()
const dialog = useDialog()

const searchQuery = ref('')
const showNewModal = ref(false)
const showDetailModal = ref(false)
const selectedRecord = ref<BoardRecord | null>(null)

const filteredRecords = computed(() => store.searchBoardRecords(searchQuery.value))

function handleView(id: string) {
  const record = store.boardRecords.find((r) => r.id === id)
  if (record) {
    selectedRecord.value = record
    showDetailModal.value = true
  }
}

function handleDelete(id: string) {
  dialog.warning({
    title: 'Delete Record',
    content: 'Are you sure you want to delete this board discussion record?',
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: () => {
      store.deleteBoardRecord(id)
    },
  })
}
</script>

<template>
  <div>
    <PageHeader title="Board Records">
      <template #actions>
        <NButton type="primary" @click="showNewModal = true">
          <template #icon><NIcon :component="AddOutline" /></template>
          New Discussion
        </NButton>
      </template>
    </PageHeader>

    <NInput
      v-model:value="searchQuery"
      placeholder="Search by topic or synthesis..."
      clearable
      style="margin-bottom: 16px"
    >
      <template #prefix>
        <NIcon :component="SearchOutline" />
      </template>
    </NInput>

    <BoardRecordList
      :records="filteredRecords"
      @view="handleView"
      @delete="handleDelete"
    />

    <NewDiscussionModal v-model:show="showNewModal" @created="searchQuery = ''" />
    <BoardRecordDetail v-model:show="showDetailModal" :record="selectedRecord" />
  </div>
</template>
