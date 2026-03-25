<script setup lang="ts">
import { ref } from 'vue'
import {
  NForm, NFormItem, NInput, NInputNumber, NSelect, NButton,
  NSpace, NDivider, type FormRules, type FormInst,
} from 'naive-ui'
import type { SetupOrgRequest } from '@/types'

const emit = defineEmits<{ submit: [data: SetupOrgRequest] }>()
const formRef = ref<FormInst | null>(null)

const form = ref<SetupOrgRequest>({
  industry: '',
  company_stage: '',
  business_model: '',
  team_size: 10,
  org_structure: '',
  current_projects: '',
  pain_points: [],
  comm_tools: [],
  culture_prefs: '',
  goal_framework: '',
})

const stageOptions = [
  { label: 'Startup', value: 'Startup' },
  { label: 'Growth', value: 'Growth' },
  { label: 'Mature', value: 'Mature' },
]

const modelOptions = [
  { label: 'B2B', value: 'B2B' },
  { label: 'B2C', value: 'B2C' },
  { label: 'Marketplace', value: 'Marketplace' },
  { label: 'SaaS', value: 'SaaS' },
]

const structureOptions = [
  { label: 'Flat', value: 'Flat' },
  { label: 'Hierarchical', value: 'Hierarchical' },
  { label: 'Matrix', value: 'Matrix' },
  { label: 'Team-based', value: 'Team-based' },
]

const painPointOptions = [
  { label: 'Poor communication', value: 'Poor communication' },
  { label: 'No metrics', value: 'No metrics' },
  { label: 'Low engagement', value: 'Low engagement' },
  { label: 'Unclear roles', value: 'Unclear roles' },
  { label: 'Slow decisions', value: 'Slow decisions' },
  { label: 'High turnover', value: 'High turnover' },
]

const commToolOptions = [
  { label: 'Telegram', value: 'Telegram' },
  { label: 'Slack', value: 'Slack' },
  { label: 'Lark', value: 'Lark' },
  { label: 'Email', value: 'Email' },
  { label: 'WhatsApp', value: 'WhatsApp' },
]

const frameworkOptions = [
  { label: 'OKR', value: 'OKR' },
  { label: 'KPI', value: 'KPI' },
  { label: 'Scrum', value: 'Scrum' },
  { label: 'MBO', value: 'MBO' },
  { label: 'BSC', value: 'BSC' },
]

const rules: FormRules = {
  industry: { required: true, message: 'Industry is required', trigger: 'blur' },
  company_stage: { required: true, message: 'Company stage is required', trigger: 'change' },
  team_size: { required: true, type: 'number', min: 1, message: 'Team size must be at least 1', trigger: 'blur' },
  org_structure: { required: true, message: 'Org structure is required', trigger: 'change' },
  pain_points: { required: true, type: 'array', min: 1, message: 'Select at least 1 pain point', trigger: 'change' },
  comm_tools: { required: true, type: 'array', min: 1, message: 'Select at least 1 tool', trigger: 'change' },
}

function handleSubmit() {
  formRef.value?.validate((errors) => {
    if (!errors) {
      emit('submit', { ...form.value })
    }
  })
}
</script>

<template>
  <NForm ref="formRef" :model="form" :rules="rules" label-placement="top" style="max-width: 640px">
    <NDivider title-placement="left" style="margin-top: 0">Basic Info</NDivider>

    <NFormItem label="Industry" path="industry">
      <NInput v-model:value="form.industry" placeholder="e.g. Tech, Manufacturing, Finance" />
    </NFormItem>

    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px">
      <NFormItem label="Company Stage" path="company_stage">
        <NSelect v-model:value="form.company_stage" :options="stageOptions" placeholder="Select" />
      </NFormItem>
      <NFormItem label="Business Model" path="business_model">
        <NSelect v-model:value="form.business_model" :options="modelOptions" placeholder="Optional" clearable />
      </NFormItem>
    </div>

    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px">
      <NFormItem label="Team Size" path="team_size">
        <NInputNumber v-model:value="form.team_size" :min="1" :max="100000" style="width: 100%" />
      </NFormItem>
      <NFormItem label="Org Structure" path="org_structure">
        <NSelect v-model:value="form.org_structure" :options="structureOptions" placeholder="Select" />
      </NFormItem>
    </div>

    <NDivider title-placement="left">Organization</NDivider>

    <NFormItem label="Current Projects" path="current_projects">
      <NInput v-model:value="form.current_projects" type="textarea" :rows="2" placeholder="Brief description of your team's current work" />
    </NFormItem>

    <NDivider title-placement="left">Preferences</NDivider>

    <NFormItem label="Pain Points" path="pain_points">
      <NSelect v-model:value="form.pain_points" :options="painPointOptions" multiple placeholder="Select pain points" />
    </NFormItem>

    <NFormItem label="Communication Tools" path="comm_tools">
      <NSelect v-model:value="form.comm_tools" :options="commToolOptions" multiple placeholder="Select tools your team uses" />
    </NFormItem>

    <NFormItem label="Culture Preferences" path="culture_prefs">
      <NInput v-model:value="form.culture_prefs" type="textarea" :rows="2" placeholder="Optional: describe your ideal team culture" />
    </NFormItem>

    <NFormItem label="Goal Framework" path="goal_framework">
      <NSelect v-model:value="form.goal_framework" :options="frameworkOptions" placeholder="Optional" clearable />
    </NFormItem>

    <NSpace justify="end" style="margin-top: 16px">
      <NButton type="primary" @click="handleSubmit">Generate Plan</NButton>
    </NSpace>
  </NForm>
</template>
