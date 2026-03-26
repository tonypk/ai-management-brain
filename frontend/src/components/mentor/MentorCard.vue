<script setup lang="ts">
import { NCard, NTag, NButton, NSpace, NEllipsis } from 'naive-ui'
import type { MentorWithDomain } from '@/types'

defineProps<{
  mentor: MentorWithDomain
  isActive: boolean
  switching: boolean
}>()

const emit = defineEmits<{
  switch: [mentorId: string]
}>()

const mentorEmojis: Record<string, string> = {
  musk: '🚀',
  inamori: '🙏',
  dalio: '📊',
  grove: '🔧',
  ren: '🐺',
  son: '🌐',
  jobs: '🎨',
  bezos: '📦',
  ma: '🛒',
}
</script>

<template>
  <NCard
    :bordered="true"
    hoverable
    :style="{
      borderColor: isActive ? '#18a058' : undefined,
      borderWidth: isActive ? '2px' : undefined,
    }"
    content-style="padding: 16px"
  >
    <div style="text-align: center">
      <div style="font-size: 32px; margin-bottom: 8px">
        {{ mentorEmojis[mentor.id] || '🧠' }}
      </div>
      <div style="font-weight: 600; font-size: 15px; margin-bottom: 4px">
        {{ mentor.name_en || mentor.name }}
      </div>
      <div v-if="mentor.company" style="color: #999; font-size: 12px; margin-bottom: 8px">
        {{ mentor.company }}
      </div>
      <NEllipsis :line-clamp="2" :tooltip="{ width: 280 }">
        <div style="color: #666; font-size: 13px; margin-bottom: 8px">
          {{ mentor.philosophy }}
        </div>
      </NEllipsis>
      <NSpace justify="center" :size="4" style="margin-bottom: 12px">
        <NTag v-if="mentor.domain" size="small" :bordered="false" type="info">
          {{ mentor.domain }}
        </NTag>
      </NSpace>
      <NSpace justify="center" :size="4" :wrap="true" style="margin-bottom: 12px">
        <NTag v-for="tag in (mentor.tags || []).slice(0, 3)" :key="tag" size="tiny" :bordered="false">
          {{ tag }}
        </NTag>
      </NSpace>
      <NButton
        v-if="isActive"
        type="success"
        size="small"
        disabled
        block
      >
        Active
      </NButton>
      <NButton
        v-else
        size="small"
        block
        :loading="switching"
        @click="emit('switch', mentor.id)"
      >
        Switch
      </NButton>
    </div>
  </NCard>
</template>
