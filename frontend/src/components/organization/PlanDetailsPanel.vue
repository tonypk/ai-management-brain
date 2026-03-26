<script setup lang="ts">
import { NCollapse, NCollapseItem, NDataTable, NTag, type DataTableColumns } from 'naive-ui'
import type { ManagementPlan, KpiItem, MeetingItem, AlertRule } from '@/types'

defineProps<{ plan: ManagementPlan }>()

const kpiColumns: DataTableColumns<KpiItem> = [
  { title: 'Name', key: 'name' },
  { title: 'Target', key: 'target' },
  { title: 'Frequency', key: 'frequency' },
  { title: 'Owner', key: 'owner' },
]

const meetingColumns: DataTableColumns<MeetingItem> = [
  { title: 'Name', key: 'name' },
  { title: 'Frequency', key: 'frequency' },
  { title: 'Duration', key: 'duration' },
  { title: 'Attendees', key: 'attendees' },
  { title: 'Purpose', key: 'purpose' },
]

const alertColumns: DataTableColumns<AlertRule> = [
  { title: 'Condition', key: 'condition' },
  { title: 'Action', key: 'action' },
  { title: 'Message', key: 'message' },
]
</script>

<template>
  <NCollapse :default-expanded-names="['framework']">
    <NCollapseItem title="Framework" name="framework">
      <div style="padding: 8px 0">
        <div style="font-weight: 600; margin-bottom: 8px">{{ plan.management_framework }}</div>
        <div style="color: #666; white-space: pre-wrap">{{ plan.reasoning }}</div>
      </div>
    </NCollapseItem>

    <NCollapseItem title="Culture Principles" name="culture">
      <div style="padding: 8px 0">
        <div v-for="(p, i) in plan.culture_principles" :key="i" style="display: flex; gap: 8px; margin-bottom: 6px">
          <NTag size="small" type="info" round>{{ i + 1 }}</NTag>
          <span>{{ p }}</span>
        </div>
      </div>
    </NCollapseItem>

    <NCollapseItem title="KPI System" name="kpi">
      <NDataTable
        :columns="kpiColumns"
        :data="plan.kpi_system"
        :bordered="false"
        size="small"
        :pagination="false"
      />
    </NCollapseItem>

    <NCollapseItem title="Meeting Cadence" name="meetings">
      <NDataTable
        :columns="meetingColumns"
        :data="plan.meeting_cadence"
        :bordered="false"
        size="small"
        :pagination="false"
      />
    </NCollapseItem>

    <NCollapseItem title="Alert Rules" name="alerts">
      <NDataTable
        :columns="alertColumns"
        :data="plan.alert_rules"
        :bordered="false"
        size="small"
        :pagination="false"
      />
    </NCollapseItem>

    <NCollapseItem title="Daily Questions" name="questions">
      <div v-for="(questions, role) in plan.daily_questions" :key="role" style="margin-bottom: 16px">
        <div style="font-weight: 600; margin-bottom: 6px">{{ role }}</div>
        <div v-for="(q, i) in questions" :key="i" style="padding: 4px 0 4px 16px; color: #555">
          {{ i + 1 }}. {{ q }}
        </div>
      </div>
    </NCollapseItem>
  </NCollapse>
</template>
