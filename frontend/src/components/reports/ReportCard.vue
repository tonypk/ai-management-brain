<script setup lang="ts">
import { NCard, NCollapse, NCollapseItem, NSpace, NTag } from 'naive-ui'
import SentimentBadge from '@/components/shared/SentimentBadge.vue'
import type { Report } from '@/types'

defineProps<{
  report: Report
}>()
</script>

<template>
  <NCard :bordered="false" size="small" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08); margin-bottom: 12px">
    <NCollapse>
      <NCollapseItem :title="report.employee_name" :name="report.id">
        <template #header-extra>
          <NSpace :size="8" align="center">
            <SentimentBadge :sentiment="report.sentiment" />
            <NTag v-if="report.blockers" type="error" size="small">Blocked</NTag>
            <span style="font-size: 12px; color: #888">{{ report.submitted_at }}</span>
          </NSpace>
        </template>
        <div v-if="report.answers && typeof report.answers === 'object'">
          <div v-for="(answer, question) in report.answers" :key="String(question)" style="margin-bottom: 12px">
            <div style="font-weight: 600; font-size: 13px; color: #555; margin-bottom: 4px">{{ question }}</div>
            <div style="font-size: 14px; line-height: 1.6">{{ answer }}</div>
          </div>
        </div>
        <div v-if="report.blockers" style="margin-top: 12px; padding: 12px; background: #fef2f2; border-radius: 8px; border-left: 3px solid #ef4444">
          <div style="font-weight: 600; font-size: 13px; color: #991b1b; margin-bottom: 4px">Blockers</div>
          <div style="font-size: 14px; color: #7f1d1d">{{ report.blockers }}</div>
        </div>
      </NCollapseItem>
    </NCollapse>
  </NCard>
</template>
