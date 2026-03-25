<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NTabs, NTabPane, NButton, NSpace, NSpin } from 'naive-ui'
import { useMessage } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import SetupWizard from '@/components/organization/SetupWizard.vue'
import OrgOverviewCards from '@/components/organization/OrgOverviewCards.vue'
import OrgDesignPanel from '@/components/organization/OrgDesignPanel.vue'
import PlanDetailsPanel from '@/components/organization/PlanDetailsPanel.vue'
import AIRoleList from '@/components/organization/AIRoleList.vue'
import SuggestionList from '@/components/organization/SuggestionList.vue'
import AdjustPlanModal from '@/components/organization/AdjustPlanModal.vue'
import {
  getOrgPlan, adjustPlan, activatePlan,
  getOrgRoles, getOrgSuggestions,
  approveSuggestion, rejectSuggestion,
} from '@/api'
import type { OrgPlan, AIRole, AISuggestion } from '@/types'

const message = useMessage()
const loading = ref(true)
const activating = ref(false)

const plan = ref<OrgPlan | null>(null)
const roles = ref<AIRole[]>([])
const suggestions = ref<AISuggestion[]>([])
const showAdjustModal = ref(false)
const activeTab = ref('overview')

async function fetchData() {
  loading.value = true
  try {
    plan.value = await getOrgPlan()
    const [r, s] = await Promise.all([
      getOrgRoles().catch(() => [] as AIRole[]),
      getOrgSuggestions().catch(() => [] as AISuggestion[]),
    ])
    roles.value = r
    suggestions.value = s
  } catch {
    plan.value = null
  } finally {
    loading.value = false
  }
}

async function handleAdjust(feedback: string) {
  try {
    const result = await adjustPlan(feedback)
    if (plan.value) {
      plan.value = { ...plan.value, plan: result.plan, plan_version: result.plan_version }
    }
    showAdjustModal.value = false
    message.success('Plan adjusted successfully')
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to adjust plan')
  }
}

async function handleActivate() {
  activating.value = true
  try {
    const result = await activatePlan()
    message.success(`Plan activated! ${result.roles_activated} AI roles created.`)
    await fetchData()
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to activate plan')
  } finally {
    activating.value = false
  }
}

async function handleApprove(id: string) {
  try {
    await approveSuggestion(id)
    suggestions.value = suggestions.value.map(s =>
      s.id === id ? { ...s, status: 'approved' as const, reviewed_at: new Date().toISOString() } : s
    )
    message.success('Suggestion approved')
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to approve')
  }
}

async function handleReject(id: string) {
  try {
    await rejectSuggestion(id)
    suggestions.value = suggestions.value.map(s =>
      s.id === id ? { ...s, status: 'rejected' as const, reviewed_at: new Date().toISOString() } : s
    )
    message.success('Suggestion rejected')
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to reject')
  }
}

onMounted(fetchData)
</script>

<template>
  <div>
    <PageHeader title="Organization">
      <template #actions>
        <NSpace v-if="plan">
          <NButton @click="showAdjustModal = true">Adjust Plan</NButton>
          <NButton
            v-if="plan.status !== 'active'"
            type="primary"
            :loading="activating"
            @click="handleActivate"
          >
            Activate
          </NButton>
        </NSpace>
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <template v-if="plan">
        <NTabs v-model:value="activeTab" type="line" style="margin-bottom: 16px">
          <NTabPane name="overview" tab="Overview">
            <OrgOverviewCards :plan="plan" style="margin-bottom: 16px" />
            <OrgDesignPanel :design="plan.plan.org_design" />
          </NTabPane>

          <NTabPane name="details" tab="Plan Details">
            <PlanDetailsPanel :plan="plan.plan" />
          </NTabPane>

          <NTabPane name="roles" tab="AI Roles">
            <AIRoleList :roles="roles" />
          </NTabPane>

          <NTabPane name="suggestions" tab="Suggestions">
            <SuggestionList
              :suggestions="suggestions"
              @approve="handleApprove"
              @reject="handleReject"
            />
          </NTabPane>
        </NTabs>
      </template>

      <SetupWizard v-else-if="!loading" @complete="fetchData" />
    </NSpin>

    <AdjustPlanModal
      v-model:show="showAdjustModal"
      @submit="handleAdjust"
    />
  </div>
</template>
