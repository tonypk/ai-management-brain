<script setup lang="ts">
import { NGrid, NGi, NCard, NTag, NSpace } from 'naive-ui'
import { useRouter } from 'vue-router'
import SentimentBadge from '@/components/shared/SentimentBadge.vue'
import type { EmployeeActivity } from '@/types'

defineProps<{
  employees: EmployeeActivity[]
}>()

const router = useRouter()

const sentimentEmoji: Record<string, string> = {
  positive: '😊',
  neutral: '😐',
  negative: '😟',
  mixed: '😕',
}

function navigateToProfile(id: string): void {
  router.push(`/employees/${id}`)
}
</script>

<template>
  <NCard title="Employee Sentiment" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <NGrid :cols="24" :x-gap="12" :y-gap="12" responsive="screen" :item-responsive="true">
      <NGi v-for="emp in employees" :key="emp.id" span="24 s:12 m:8 l:6">
        <NCard
          size="small"
          hoverable
          style="cursor: pointer"
          @click="navigateToProfile(emp.id)"
        >
          <div style="text-align: center">
            <div style="font-size: 32px; margin-bottom: 4px">
              {{ sentimentEmoji[emp.last_sentiment] || '❓' }}
            </div>
            <div style="font-weight: 600; font-size: 14px; margin-bottom: 4px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              {{ emp.name }}
            </div>
            <NSpace justify="center" :size="4" style="margin-bottom: 6px">
              <SentimentBadge :sentiment="emp.last_sentiment" />
            </NSpace>
            <div style="font-size: 12px; color: #888">
              <span style="color: #22c55e">{{ emp.submitted_7d }}/7</span>
              <NTag v-if="emp.missed_7d >= 3" type="warning" size="tiny" round style="margin-left: 4px">
                {{ emp.missed_7d }} missed
              </NTag>
            </div>
          </div>
        </NCard>
      </NGi>
    </NGrid>
  </NCard>
</template>
