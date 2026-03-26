<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NGrid, NGi, NSpin } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import AtRiskList from '@/components/coaching/AtRiskList.vue'
import TalkingPointsCard from '@/components/coaching/TalkingPointsCard.vue'
import CoachingChatPanel from '@/components/coaching/CoachingChatPanel.vue'
import { getEmployeeActivity } from '@/api/dashboard'
import { getAlerts } from '@/api/alerts'
import type { EmployeeActivity, Alert, AtRiskEmployee, TalkingPoint, RiskLevel } from '@/types'

const loading = ref(true)
const selectedEmployeeId = ref<string>()
const activity = ref<EmployeeActivity[]>([])
const alerts = ref<Alert[]>([])

function assessRisk(emp: EmployeeActivity, empAlerts: Alert[]): RiskLevel {
  const hasCriticalAlert = empAlerts.some(a => a.severity === 'critical')
  if (hasCriticalAlert || emp.missed_7d >= 5) return 'high'
  if (emp.missed_7d >= 3 || emp.last_sentiment === 'negative') return 'medium'
  if (emp.last_sentiment === 'mixed') return 'low'
  return 'none'
}

const atRiskEmployees = computed<AtRiskEmployee[]>(() => {
  return activity.value
    .map(emp => {
      const empAlerts = alerts.value.filter(a => a.employee_id === emp.id)
      return {
        id: emp.id,
        name: emp.name,
        risk: assessRisk(emp, empAlerts),
        missed_7d: emp.missed_7d,
        last_sentiment: emp.last_sentiment,
        culture_code: emp.culture_code,
      }
    })
    .filter(e => e.risk !== 'none')
    .sort((a, b) => {
      const order: Record<RiskLevel, number> = { high: 0, medium: 1, low: 2, none: 3 }
      return order[a.risk] - order[b.risk]
    })
})

const talkingPoints = computed<TalkingPoint[]>(() => {
  return atRiskEmployees.value.map(emp => {
    const points: string[] = []
    if (emp.missed_7d >= 5) {
      points.push(`Missed ${emp.missed_7d} days in the last week (HIGH concern)`)
    } else if (emp.missed_7d >= 3) {
      points.push(`Missed ${emp.missed_7d} days in the last week`)
    }
    if (emp.last_sentiment === 'negative') {
      points.push('Recent sentiment is negative — explore underlying concerns')
    } else if (emp.last_sentiment === 'mixed') {
      points.push('Sentiment is mixed — check if there are unspoken issues')
    }
    const empAlerts = alerts.value.filter(a => a.employee_id === emp.id)
    for (const alert of empAlerts) {
      points.push(alert.message)
    }
    if (points.length === 0) {
      points.push('Follow up on recent trends')
    }
    return {
      employee_name: emp.name,
      priority: emp.risk,
      points,
    }
  })
})

function handleSelectEmployee(emp: AtRiskEmployee): void {
  selectedEmployeeId.value = emp.id
}

onMounted(async () => {
  try {
    const [act, al] = await Promise.all([
      getEmployeeActivity(),
      getAlerts(),
    ])
    activity.value = act
    alerts.value = al
  } catch {
    // Components handle missing data gracefully
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <PageHeader title="1:1 Coaching Assistant" />

    <NSpin :show="loading">
      <NGrid :cols="24" :x-gap="16" :y-gap="16" responsive="screen" :item-responsive="true">
        <!-- Left column: Risk list + Talking points -->
        <NGi span="24 m:14">
          <AtRiskList
            :employees="atRiskEmployees"
            :selected-id="selectedEmployeeId"
            @select="handleSelectEmployee"
          />
          <div style="margin-top: 16px">
            <TalkingPointsCard :points="talkingPoints" />
          </div>
        </NGi>

        <!-- Right column: Chat -->
        <NGi span="24 m:10">
          <CoachingChatPanel
            :employees="atRiskEmployees"
            :selected-id="selectedEmployeeId"
          />
        </NGi>
      </NGrid>
    </NSpin>
  </div>
</template>
