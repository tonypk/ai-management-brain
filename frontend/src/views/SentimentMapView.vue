<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NGrid, NGi, NSpin } from 'naive-ui'
import { PeopleOutline, SpeedometerOutline, HappyOutline } from '@vicons/ionicons5'
import PageHeader from '@/components/shared/PageHeader.vue'
import StatCard from '@/components/shared/StatCard.vue'
import HealthGauge from '@/components/shared/HealthGauge.vue'
import SubmissionTrendChart from '@/components/dashboard/SubmissionTrendChart.vue'
import SentimentDistributionChart from '@/components/sentiment/SentimentDistributionChart.vue'
import EmployeeSentimentGrid from '@/components/sentiment/EmployeeSentimentGrid.vue'
import { getAnalyticsOverview, getEmployeeActivity } from '@/api/dashboard'
import type { AnalyticsOverview, EmployeeActivity } from '@/types'

const loading = ref(true)
const analytics = ref<AnalyticsOverview | null>(null)
const activity = ref<EmployeeActivity[]>([])

const positiveRate = computed(() => {
  if (!analytics.value?.sentiment) return '0%'
  const s = analytics.value.sentiment
  const total = Object.values(s).reduce((sum, v) => sum + v, 0)
  if (total === 0) return '0%'
  return Math.round(((s.positive || 0) / total) * 100) + '%'
})

const submissionRate = computed(() => {
  if (!analytics.value?.today) return '0%'
  return Math.round(analytics.value.today.submission_rate * 100) + '%'
})

onMounted(async () => {
  try {
    const [a, act] = await Promise.all([
      getAnalyticsOverview(),
      getEmployeeActivity(),
    ])
    analytics.value = a
    activity.value = act
  } catch {
    // Components handle missing data gracefully
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <PageHeader title="Sentiment Map" />

    <NSpin :show="loading">
      <!-- Row 1: Stat cards -->
      <NGrid :cols="24" :x-gap="16" :y-gap="16" responsive="screen" :item-responsive="true">
        <NGi span="24 m:6">
          <HealthGauge :value="analytics?.health_score ?? 0" title="Health" />
        </NGi>
        <NGi span="24 m:6">
          <StatCard
            label="Employees"
            :value="analytics?.today?.employees ?? 0"
            :icon="PeopleOutline"
            icon-color="#6366f1"
          />
        </NGi>
        <NGi span="24 m:6">
          <StatCard
            label="Submission Rate"
            :value="submissionRate"
            :icon="SpeedometerOutline"
            icon-color="#f59e0b"
          />
        </NGi>
        <NGi span="24 m:6">
          <StatCard
            label="Positive"
            :value="positiveRate"
            :icon="HappyOutline"
            icon-color="#22c55e"
          />
        </NGi>
      </NGrid>

      <!-- Row 2: Pie chart + Trend -->
      <NGrid :cols="24" :x-gap="16" :y-gap="16" style="margin-top: 16px" responsive="screen" :item-responsive="true">
        <NGi span="24 m:10">
          <SentimentDistributionChart :data="analytics?.sentiment ?? {}" />
        </NGi>
        <NGi span="24 m:14">
          <SubmissionTrendChart :data="analytics?.trend_7d ?? []" />
        </NGi>
      </NGrid>

      <!-- Row 3: Employee grid -->
      <div style="margin-top: 16px">
        <EmployeeSentimentGrid :employees="activity" />
      </div>
    </NSpin>
  </div>
</template>
