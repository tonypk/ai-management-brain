<script setup lang="ts">
import { NCard, NProgress, NTag, NIcon } from 'naive-ui'
import { CheckmarkCircleOutline, TimeOutline, CloseCircleOutline } from '@vicons/ionicons5'
import type { CheckinStatus } from '@/types'

const props = defineProps<{
  status: CheckinStatus | null
}>()
</script>

<template>
  <NCard title="Today's Check-ins" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <template v-if="status">
      <NProgress
        type="line"
        :percentage="status.total_employees ? Math.round((status.submitted.length / status.total_employees) * 100) : 0"
        :height="20"
        :border-radius="10"
        indicator-placement="inside"
        processing
      />
      <div style="margin-top: 16px; display: flex; flex-wrap: wrap; gap: 8px">
        <NTag
          v-for="emp in status.submitted"
          :key="emp.id"
          type="success"
          size="medium"
          round
        >
          <template #icon>
            <NIcon :component="CheckmarkCircleOutline" />
          </template>
          {{ emp.name }}
        </NTag>
        <NTag
          v-for="emp in status.pending"
          :key="emp.id"
          type="warning"
          size="medium"
          round
        >
          <template #icon>
            <NIcon :component="TimeOutline" />
          </template>
          {{ emp.name }}
          <span v-if="emp.chase_count > 0" style="margin-left: 4px; opacity: 0.7">
            (chased {{ emp.chase_count }}x)
          </span>
        </NTag>
        <NTag
          v-for="name in status.missed"
          :key="name"
          type="error"
          size="medium"
          round
        >
          <template #icon>
            <NIcon :component="CloseCircleOutline" />
          </template>
          {{ name }}
        </NTag>
      </div>
      <div style="margin-top: 12px; font-size: 13px; color: #888">
        {{ status.submitted.length }} submitted / {{ status.pending.length }} pending / {{ status.missed.length }} missed
        of {{ status.total_employees }} total
      </div>
    </template>
  </NCard>
</template>
