<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  NButton, NIcon, NSpin, NDataTable, NTag, NTabs, NTabPane,
  NModal, NInput, NSelect, NSpace, NFormItem,
  useMessage,
} from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import { listIncentiveRules, createIncentiveRule, deleteIncentiveRule, listIncentiveScores } from '@/api/incentives'
import type { IncentiveRule, IncentiveScore } from '@/types'

const message = useMessage()

const loading = ref(true)
const rules = ref<IncentiveRule[]>([])
const scores = ref<IncentiveScore[]>([])
const scorePeriod = ref(new Date().toISOString().slice(0, 7)) // YYYY-MM

const showCreateModal = ref(false)
const form = ref({
  name: '', reward_model: 'bonus', payout_cycle: 'monthly',
})

onMounted(async () => {
  try {
    const [r, s] = await Promise.all([
      listIncentiveRules(),
      listIncentiveScores(scorePeriod.value),
    ])
    rules.value = r
    scores.value = s
  } catch {
    message.error('Failed to load incentives')
  } finally {
    loading.value = false
  }
})

async function handleCreate() {
  try {
    const r = await createIncentiveRule(form.value)
    rules.value.unshift(r)
    showCreateModal.value = false
    form.value = { name: '', reward_model: 'bonus', payout_cycle: 'monthly' }
    message.success('Rule created')
  } catch {
    message.error('Failed to create rule')
  }
}

async function handleDelete(id: string) {
  try {
    await deleteIncentiveRule(id)
    rules.value = rules.value.filter((r) => r.id !== id)
    message.success('Rule deleted')
  } catch {
    message.error('Failed to delete rule')
  }
}

async function loadScores() {
  try {
    scores.value = await listIncentiveScores(scorePeriod.value)
  } catch {
    message.error('Failed to load scores')
  }
}

const rewardModelOptions = [
  { label: 'Bonus', value: 'bonus' },
  { label: 'Commission', value: 'commission' },
  { label: 'Points', value: 'points' },
  { label: 'Recognition', value: 'recognition' },
]

const payoutCycleOptions = [
  { label: 'Monthly', value: 'monthly' },
  { label: 'Quarterly', value: 'quarterly' },
  { label: 'Annually', value: 'annually' },
]

const ruleColumns: DataTableColumns<IncentiveRule> = [
  { title: 'Name', key: 'name' },
  { title: 'Reward Model', key: 'reward_model', width: 110 },
  { title: 'Payout Cycle', key: 'payout_cycle', width: 110 },
  {
    title: 'Active', key: 'is_active', width: 80,
    render: (r) => h(NTag, { size: 'small', type: r.is_active ? 'success' : 'default' }, () => r.is_active ? 'Yes' : 'No'),
  },
  {
    title: '', key: 'action', width: 60,
    render: (r) => h(NButton, { size: 'tiny', type: 'error', quaternary: true, onClick: () => handleDelete(r.id) }, () => 'Del'),
  },
]

const scoreColumns: DataTableColumns<IncentiveScore> = [
  { title: 'Employee', key: 'person_name', render: (s) => s.person_name || s.person_id },
  { title: 'Rule', key: 'rule_name', render: (s) => s.rule_name || s.rule_id },
  { title: 'Score', key: 'score', width: 80 },
  { title: 'Weight', key: 'payout_weight', width: 80 },
  { title: 'Confidence', key: 'attribution_confidence', width: 100 },
  {
    title: 'Status', key: 'status', width: 90,
    render: (s) => h(NTag, { size: 'small', type: s.status === 'finalized' ? 'success' : 'warning' }, () => s.status),
  },
]
</script>

<script lang="ts">
import { h } from 'vue'
export default {}
</script>

<template>
  <div>
    <PageHeader title="Incentives">
      <template #actions>
        <NButton type="primary" @click="showCreateModal = true">
          <template #icon><NIcon :component="AddOutline" /></template>
          New Rule
        </NButton>
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <NTabs type="line">
        <NTabPane name="rules" tab="Rules">
          <EmptyState v-if="rules.length === 0 && !loading" description="No incentive rules defined yet" />
          <NDataTable v-else :columns="ruleColumns" :data="rules" :bordered="false" size="small" />
        </NTabPane>
        <NTabPane name="scores" tab="Scores">
          <NSpace :size="8" align="center" style="margin-bottom: 12px">
            <NInput v-model:value="scorePeriod" placeholder="YYYY-MM" style="width: 140px" />
            <NButton @click="loadScores">Load</NButton>
          </NSpace>
          <EmptyState v-if="scores.length === 0" description="No scores for this period" />
          <NDataTable v-else :columns="scoreColumns" :data="scores" :bordered="false" size="small" />
        </NTabPane>
      </NTabs>
    </NSpin>

    <!-- Create Rule Modal -->
    <NModal v-model:show="showCreateModal" preset="card" title="New Incentive Rule" style="max-width: 420px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Name" :show-feedback="false">
          <NInput v-model:value="form.name" placeholder="e.g. Sales Commission Q2" />
        </NFormItem>
        <NFormItem label="Reward Model" :show-feedback="false">
          <NSelect v-model:value="form.reward_model" :options="rewardModelOptions" />
        </NFormItem>
        <NFormItem label="Payout Cycle" :show-feedback="false">
          <NSelect v-model:value="form.payout_cycle" :options="payoutCycleOptions" />
        </NFormItem>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCreateModal = false">Cancel</NButton>
          <NButton type="primary" :disabled="!form.name.trim()" @click="handleCreate">Create</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
