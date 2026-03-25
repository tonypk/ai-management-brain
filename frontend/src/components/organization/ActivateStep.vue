<script setup lang="ts">
import { NButton, NSpace, NCard, NIcon } from 'naive-ui'
import { CheckmarkCircleOutline as CheckIcon } from '@vicons/ionicons5'

defineProps<{ activating: boolean }>()
const emit = defineEmits<{ activate: []; back: [] }>()
</script>

<template>
  <NCard :bordered="false" style="max-width: 480px; box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <div style="text-align: center; padding: 24px 0">
      <NIcon :size="48" color="#22c55e" :component="CheckIcon" />
      <h3 style="margin: 16px 0 8px">Ready to Activate</h3>
      <p style="color: #666; margin-bottom: 24px">
        This will create AI roles and start automated management based on your plan.
        You can adjust the plan later from the Organization page.
      </p>
      <NSpace justify="center">
        <NButton @click="emit('back')">Back</NButton>
        <NButton type="primary" :loading="activating" @click="emit('activate')">
          Activate Plan
        </NButton>
      </NSpace>
    </div>
  </NCard>
</template>
