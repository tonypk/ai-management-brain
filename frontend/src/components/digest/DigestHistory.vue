<script setup lang="ts">
import { NCard, NCollapse, NCollapseItem, NButton, NIcon, NSpace, NText, NTag, useMessage } from 'naive-ui'
import { TrashOutline } from '@vicons/ionicons5'
import EmptyState from '@/components/shared/EmptyState.vue'
import type { DigestRecord } from '@/types'

defineProps<{
  records: DigestRecord[]
}>()

const emit = defineEmits<{
  delete: [id: string]
}>()

const message = useMessage()

function preview(content: string): string {
  const first = content.split('\n').find((l) => l.trim() && !l.startsWith('#'))
  return first ? first.slice(0, 80) + (first.length > 80 ? '...' : '') : ''
}

async function handleCopy(content: string) {
  try {
    await navigator.clipboard.writeText(content)
    message.success('Copied to clipboard')
  } catch {
    message.error('Failed to copy')
  }
}
</script>

<template>
  <NCard
    :bordered="false"
    style="box-shadow: 0 1px 3px rgba(0,0,0,0.08); margin-top: 16px"
  >
    <template #header>
      <span style="font-weight: 600">Past Digests</span>
    </template>

    <EmptyState v-if="records.length === 0" description="No past digests" />

    <NCollapse v-else>
      <NCollapseItem
        v-for="record in records"
        :key="record.id"
        :name="record.id"
      >
        <template #header>
          <NSpace align="center" :size="8">
            <NTag size="small" :bordered="false" :type="record.period === 'weekly' ? 'info' : 'success'">
              {{ record.period }}
            </NTag>
            <NText style="font-size: 13px; font-weight: 500">{{ record.period_label }}</NText>
            <NText depth="3" style="font-size: 12px">{{ preview(record.content) }}</NText>
          </NSpace>
        </template>
        <div style="white-space: pre-wrap; line-height: 1.7; font-size: 14px; padding: 8px 0">
          {{ record.content }}
        </div>
        <NSpace :size="8" style="margin-top: 8px">
          <NButton size="tiny" quaternary @click.stop="handleCopy(record.content)">Copy</NButton>
          <NButton size="tiny" quaternary type="error" @click.stop="emit('delete', record.id)">
            <template #icon><NIcon :component="TrashOutline" :size="14" /></template>
            Delete
          </NButton>
        </NSpace>
      </NCollapseItem>
    </NCollapse>
  </NCard>
</template>
