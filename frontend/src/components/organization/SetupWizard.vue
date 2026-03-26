<script setup lang="ts">
import { ref } from 'vue'
import { NSteps, NStep, NSpin, NAlert, NButton } from 'naive-ui'
import { useMessage } from 'naive-ui'
import CompanyProfileForm from './CompanyProfileForm.vue'
import PlanReviewPanel from './PlanReviewPanel.vue'
import ActivateStep from './ActivateStep.vue'
import { setupOrg, adjustPlan, activatePlan } from '@/api'
import type { SetupOrgRequest, OrgPlan, ManagementPlan } from '@/types'

const emit = defineEmits<{ complete: [] }>()
const message = useMessage()

const currentStep = ref(1)
const loading = ref(false)
const adjusting = ref(false)
const activating = ref(false)
const error = ref('')

const savedFormData = ref<SetupOrgRequest | null>(null)
const orgPlan = ref<OrgPlan | null>(null)
const plan = ref<ManagementPlan | null>(null)

async function handleProfileSubmit(data: SetupOrgRequest) {
  savedFormData.value = data
  loading.value = true
  error.value = ''
  try {
    orgPlan.value = await setupOrg(data)
    plan.value = orgPlan.value.plan
    currentStep.value = 2
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to generate plan'
  } finally {
    loading.value = false
  }
}

async function handleAdjust(feedback: string) {
  adjusting.value = true
  try {
    const result = await adjustPlan(feedback)
    plan.value = result.plan
    if (orgPlan.value) {
      orgPlan.value = { ...orgPlan.value, plan: result.plan, plan_version: result.plan_version }
    }
    message.success('Plan adjusted')
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to adjust plan')
  } finally {
    adjusting.value = false
  }
}

function handleConfirm() {
  currentStep.value = 3
}

async function handleActivate() {
  activating.value = true
  try {
    await activatePlan()
    message.success('Plan activated!')
    emit('complete')
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to activate')
  } finally {
    activating.value = false
  }
}

function handleBack() {
  if (currentStep.value > 1) {
    currentStep.value -= 1
  }
}

async function handleRetry() {
  if (savedFormData.value) {
    await handleProfileSubmit(savedFormData.value)
  }
}
</script>

<template>
  <div>
    <h2 style="font-size: 20px; font-weight: 700; margin-bottom: 8px">Set Up Your Organization</h2>
    <p style="color: #666; margin-bottom: 24px">
      Tell us about your company and AI will design a management plan tailored to your needs.
    </p>

    <NSteps :current="currentStep" style="margin-bottom: 32px; max-width: 600px">
      <NStep title="Company Profile" />
      <NStep title="Review Plan" />
      <NStep title="Activate" />
    </NSteps>

    <NSpin :show="loading">
      <template v-if="currentStep === 1">
        <NAlert v-if="error" type="error" style="margin-bottom: 8px; max-width: 640px" closable @close="error = ''">
          {{ error }}
        </NAlert>
        <NSpace v-if="error" style="margin-bottom: 16px">
          <NButton size="small" @click="handleRetry">Retry</NButton>
        </NSpace>
        <CompanyProfileForm @submit="handleProfileSubmit" />
      </template>

      <template v-else-if="currentStep === 2 && plan">
        <PlanReviewPanel
          :plan="plan"
          :adjusting="adjusting"
          @adjust="handleAdjust"
          @confirm="handleConfirm"
          @back="handleBack"
        />
      </template>

      <template v-else-if="currentStep === 3">
        <ActivateStep
          :activating="activating"
          @activate="handleActivate"
          @back="handleBack"
        />
      </template>
    </NSpin>
  </div>
</template>
