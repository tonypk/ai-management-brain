<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NGrid, NGi, NSpin } from 'naive-ui'
import {
  DocumentTextOutline,
  SpeedometerOutline,
  SchoolOutline,
} from '@vicons/ionicons5'
import PageHeader from '@/components/shared/PageHeader.vue'
import StatCard from '@/components/shared/StatCard.vue'
import HealthGauge from '@/components/shared/HealthGauge.vue'
import CheckinStatusPanel from '@/components/dashboard/CheckinStatusPanel.vue'
import SubmissionTrendChart from '@/components/dashboard/SubmissionTrendChart.vue'
import SentimentHeatmap from '@/components/dashboard/SentimentHeatmap.vue'
import AlertPanel from '@/components/dashboard/AlertPanel.vue'
import EmployeeActivityTable from '@/components/dashboard/EmployeeActivityTable.vue'
import { getDashboard, getAnalyticsOverview, getEmployeeActivity } from '@/api/dashboard'
import { getAlerts, getCheckinStatus } from '@/api/alerts'
import type { DashboardStats, AnalyticsOverview, EmployeeActivity, Alert, CheckinStatus } from '@/types'

const loading = ref(true)
const stats = ref<DashboardStats | null>(null)
const analytics = ref<AnalyticsOverview | null>(null)
const activity = ref<EmployeeActivity[]>([])
const alerts = ref<Alert[]>([])
const checkinStatus = ref<CheckinStatus | null>(null)

onMounted(async () => {
  try {
    const [s, a, act, al, cs] = await Promise.all([
      getDashboard(),
      getAnalyticsOverview(),
      getEmployeeActivity(),
      getAlerts(),
      getCheckinStatus(),
    ])
    stats.value = s
    analytics.value = a
    activity.value = act
    alerts.value = al
    checkinStatus.value = cs
  } catch {
    // Individual components handle missing data gracefully
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <PageHeader title="Dashboard" />

    <NSpin :show="loading">
      <!-- Row 1: Stat cards -->
      <NGrid :cols="24" :x-gap="16" :y-gap="16" responsive="screen" :item-responsive="true">
        <NGi span="24 m:6">
          <div style="height: 100%">
            <HealthGauge :value="analytics?.health_score ?? 0" title="Health" />
          </div>
        </NGi>
        <NGi span="24 m:6">
          <StatCard
            label="Today Submissions"
            :value="stats?.today_submissions ?? 0"
            :icon="DocumentTextOutline"
            icon-color="#22c55e"
          />
        </NGi>
        <NGi span="24 m:6">
          <StatCard
            label="Submission Rate"
            :value="analytics?.today?.submission_rate ? Math.round(analytics.today.submission_rate * 100) + '%' : '0%'"
            :icon="SpeedometerOutline"
            icon-color="#f59e0b"
          />
        </NGi>
        <NGi span="24 m:6">
          <StatCard
            label="Current Mentor"
            :value="stats?.current_mentor ?? '-'"
            :icon="SchoolOutline"
            icon-color="#8b5cf6"
          />
        </NGi>
      </NGrid>

      <!-- Row 2: Check-in status -->
      <div style="margin-top: 16px">
        <CheckinStatusPanel :status="checkinStatus" />
      </div>

      <!-- Row 3: Trend chart + Alert panel -->
      <NGrid :cols="24" :x-gap="16" :y-gap="16" style="margin-top: 16px" responsive="screen" :item-responsive="true">
        <NGi span="24 m:14">
          <SubmissionTrendChart :data="analytics?.trend_7d ?? []" />
        </NGi>
        <NGi span="24 m:10">
          <AlertPanel :alerts="alerts" />
        </NGi>
      </NGrid>

      <!-- Row 4: Sentiment heatmap -->
      <div style="margin-top: 16px">
        <SentimentHeatmap :employees="activity" />
      </div>

      <!-- Row 5: Employee activity table -->
      <div style="margin-top: 16px">
        <EmployeeActivityTable :data="activity" />
      </div>
    </NSpin>
  </div>
</template>
