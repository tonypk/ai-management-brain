<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NGrid, NGi, NCard, NStatistic, NSpin, NTabs, NTabPane, NDataTable } from 'naive-ui'
import {
  getWorldModelOverview,
  getWorldModelSkills,
  getWorldModelBlockers,
  getWorldModelInsights,
  type WorldModelOverview,
  type SkillRow,
  type BlockerRow,
  type InsightRow,
} from '@/api/worldmodel'
import SkillsTable from '@/components/worldmodel/SkillsTable.vue'
import InsightsPanel from '@/components/worldmodel/InsightsPanel.vue'

const loading = ref(true)
const overview = ref<WorldModelOverview | null>(null)
const skills = ref<SkillRow[]>([])
const blockers = ref<BlockerRow[]>([])
const insights = ref<InsightRow[]>([])

onMounted(async () => {
  try {
    const [o, s, b, i] = await Promise.all([
      getWorldModelOverview(),
      getWorldModelSkills(),
      getWorldModelBlockers(),
      getWorldModelInsights(),
    ])
    overview.value = o.data
    skills.value = s.data || []
    blockers.value = b.data || []
    insights.value = i.data || []
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <NSpin :show="loading">
    <NGrid :cols="24" :x-gap="16" :y-gap="16">
      <NGi :span="24">
        <NGrid :cols="24" :x-gap="12">
          <NGi span="24 m:6">
            <NCard size="small">
              <NStatistic label="Skills Tracked" :value="overview?.skill_count ?? 0" />
            </NCard>
          </NGi>
          <NGi span="24 m:6">
            <NCard size="small">
              <NStatistic label="Collaborations" :value="overview?.relationship_count ?? 0" />
            </NCard>
          </NGi>
          <NGi span="24 m:6">
            <NCard size="small">
              <NStatistic label="Active Blockers" :value="overview?.active_blocker_count ?? 0" />
            </NCard>
          </NGi>
          <NGi span="24 m:6">
            <NCard size="small">
              <NStatistic label="Growth Events (30d)" :value="overview?.growth_events_month ?? 0" />
            </NCard>
          </NGi>
        </NGrid>
      </NGi>

      <NGi :span="24">
        <NCard title="AI Insights" size="small">
          <InsightsPanel :insights="insights" />
        </NCard>
      </NGi>

      <NGi :span="24">
        <NCard size="small">
          <NTabs type="line">
            <NTabPane name="skills" tab="Team Skills">
              <SkillsTable :skills="skills" />
            </NTabPane>
            <NTabPane name="blockers" tab="Active Blockers">
              <NDataTable
                :columns="[
                  { title: 'Employee', key: 'employee_name' },
                  { title: 'Category', key: 'category' },
                  { title: 'Description', key: 'description' },
                  { title: 'Recurring', key: 'recurrence_count' },
                ]"
                :data="blockers"
                :pagination="{ pageSize: 10 }"
                size="small"
              />
            </NTabPane>
          </NTabs>
        </NCard>
      </NGi>
    </NGrid>
  </NSpin>
</template>
