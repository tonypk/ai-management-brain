<script setup lang="ts">
import { ref } from 'vue'
import { NInput, NButton, NSpace, NDivider } from 'naive-ui'
import OrgDesignPanel from './OrgDesignPanel.vue'
import PlanDetailsPanel from './PlanDetailsPanel.vue'
import type { ManagementPlan } from '@/types'

defineProps<{ plan: ManagementPlan; adjusting: boolean }>()
const emit = defineEmits<{
  adjust: [feedback: string]
  confirm: []
  back: []
}>()

const feedback = ref('')

function handleAdjust() {
  if (!feedback.value.trim()) return
  emit('adjust', feedback.value.trim())
  feedback.value = ''
}
</script>

<template>
  <div>
    <OrgDesignPanel :design="plan.org_design" />

    <NDivider />

    <PlanDetailsPanel :plan="plan" />

    <NDivider title-placement="left">Adjust</NDivider>

    <NInput
      v-model:value="feedback"
      type="textarea"
      :rows="3"
      placeholder="Optional: describe what you'd like to change..."
      :disabled="adjusting"
      style="max-width: 640px"
    />

    <NSpace style="margin-top: 16px">
      <NButton @click="emit('back')">Back</NButton>
      <NButton :loading="adjusting" :disabled="!feedback.trim()" @click="handleAdjust">
        Regenerate
      </NButton>
      <NButton type="primary" :disabled="adjusting" @click="emit('confirm')">
        Confirm Plan
      </NButton>
    </NSpace>
  </div>
</template>
