<script setup lang="ts">
import { ref, computed } from 'vue'
import { NCard, NTag, NButton, NSpace, NTabs, NTabPane } from 'naive-ui'
import type { AISuggestion } from '@/types'

const props = defineProps<{ suggestions: AISuggestion[] }>()
const emit = defineEmits<{
  approve: [id: string]
  reject: [id: string]
}>()

const activeTab = ref('pending')
const actionLoading = ref<string | null>(null)

const pending = computed(() => props.suggestions.filter(s => s.status === 'pending'))
const reviewed = computed(() => props.suggestions.filter(s => s.status !== 'pending'))

async function handleApprove(id: string) {
  actionLoading.value = id
  emit('approve', id)
}

async function handleReject(id: string) {
  actionLoading.value = id
  emit('reject', id)
}
</script>

<template>
  <NTabs v-model:value="activeTab" type="line">
    <NTabPane name="pending" :tab="`Pending (${pending.length})`">
      <div v-if="pending.length" style="display: grid; gap: 12px; margin-top: 12px">
        <NCard
          v-for="s in pending"
          :key="s.id"
          :bordered="false"
          size="small"
          style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)"
        >
          <div style="display: flex; justify-content: space-between; align-items: flex-start">
            <div style="flex: 1; min-width: 0">
              <div style="font-weight: 600">{{ s.role_id }}: {{ s.title }}</div>
              <NSpace :size="4" style="margin: 6px 0">
                <NTag size="small" type="info">{{ s.capability }}</NTag>
                <NTag size="small">{{ s.role_title }}</NTag>
              </NSpace>
              <div style="color: #555; font-size: 13px; white-space: pre-wrap; margin-top: 4px">{{ s.content }}</div>
            </div>
            <NSpace :size="8" style="margin-left: 16px; flex-shrink: 0">
              <NButton
                type="success"
                size="small"
                :loading="actionLoading === s.id"
                @click="handleApprove(s.id)"
              >
                Approve
              </NButton>
              <NButton
                size="small"
                :loading="actionLoading === s.id"
                @click="handleReject(s.id)"
              >
                Reject
              </NButton>
            </NSpace>
          </div>
        </NCard>
      </div>
      <div v-else style="color: #999; text-align: center; padding: 48px 0">No pending suggestions</div>
    </NTabPane>

    <NTabPane name="reviewed" tab="Reviewed">
      <div v-if="reviewed.length" style="display: grid; gap: 12px; margin-top: 12px">
        <NCard
          v-for="s in reviewed"
          :key="s.id"
          :bordered="false"
          size="small"
          style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)"
        >
          <div style="display: flex; justify-content: space-between; align-items: flex-start">
            <div style="flex: 1; min-width: 0">
              <div style="font-weight: 600">{{ s.role_id }}: {{ s.title }}</div>
              <NSpace :size="4" style="margin: 6px 0">
                <NTag size="small" type="info">{{ s.capability }}</NTag>
              </NSpace>
              <div style="color: #555; font-size: 13px; white-space: pre-wrap; margin-top: 4px">{{ s.content }}</div>
            </div>
            <NTag :type="s.status === 'approved' ? 'success' : 'error'" size="small" style="margin-left: 16px">
              {{ s.status }}
            </NTag>
          </div>
        </NCard>
      </div>
      <div v-else style="color: #999; text-align: center; padding: 48px 0">No reviewed suggestions</div>
    </NTabPane>
  </NTabs>
</template>
