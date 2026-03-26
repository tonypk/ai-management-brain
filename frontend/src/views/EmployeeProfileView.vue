<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { NGrid, NGi, NSpin, useMessage } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import ProfileHeader from '@/components/profile/ProfileHeader.vue'
import ActivityStatsRow from '@/components/profile/ActivityStatsRow.vue'
import ChannelInfoCard from '@/components/profile/ChannelInfoCard.vue'
import RecentReportsList from '@/components/profile/RecentReportsList.vue'
import { getEmployee, getEmployeeChannels } from '@/api/employees'
import { getEmployeeActivity } from '@/api/dashboard'
import { listReports } from '@/api/reports'
import type { Employee, EmployeeWithChannels, EmployeeActivity, Report } from '@/types'

const route = useRoute()
const message = useMessage()
const loading = ref(true)
const employee = ref<Employee | null>(null)
const channels = ref<EmployeeWithChannels | null>(null)
const activity = ref<EmployeeActivity | null>(null)
const reports = ref<Report[]>([])

function getLast7Days(): string[] {
  const dates: string[] = []
  for (let i = 0; i < 7; i++) {
    const d = new Date()
    d.setDate(d.getDate() - i)
    dates.push(d.toISOString().slice(0, 10))
  }
  return dates
}

onMounted(async () => {
  const id = route.params.id as string

  try {
    const dates = getLast7Days()

    const [emp, ch, allActivity, ...reportResults] = await Promise.all([
      getEmployee(id),
      getEmployeeChannels(id).catch(() => null),
      getEmployeeActivity(),
      ...dates.map(date => listReports(date).catch(() => [] as Report[])),
    ])

    employee.value = emp
    channels.value = ch
    activity.value = allActivity.find(a => a.id === id) ?? null

    const allReports = (reportResults as Report[][]).flat()
    reports.value = allReports
      .filter(r => r.employee_id === id)
      .sort((a, b) => b.report_date.localeCompare(a.report_date))
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Failed to load profile'
    message.error(msg)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <PageHeader
      :title="employee?.name ?? 'Employee Profile'"
      :breadcrumbs="[
        { label: 'Sentiment Map', to: '/sentiment' },
        { label: employee?.name ?? '...' },
      ]"
    />

    <NSpin :show="loading">
      <template v-if="employee">
        <!-- Profile header -->
        <ProfileHeader :employee="employee" />

        <!-- Activity stats -->
        <div style="margin-top: 16px">
          <ActivityStatsRow :activity="activity" />
        </div>

        <!-- Channels + Reports -->
        <NGrid :cols="24" :x-gap="16" :y-gap="16" style="margin-top: 16px" responsive="screen" :item-responsive="true">
          <NGi span="24 m:8">
            <ChannelInfoCard :channels="channels" />
          </NGi>
          <NGi span="24 m:16">
            <RecentReportsList :reports="reports" />
          </NGi>
        </NGrid>
      </template>
    </NSpin>
  </div>
</template>
