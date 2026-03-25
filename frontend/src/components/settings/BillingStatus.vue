<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NCard, NDescriptions, NDescriptionsItem, NTag, useMessage } from 'naive-ui'
import { getBillingStatus } from '@/api/settings'
import type { BillingStatus } from '@/types'

const message = useMessage()
const loading = ref(false)
const billing = ref<BillingStatus | null>(null)

onMounted(async () => {
  loading.value = true
  try {
    billing.value = await getBillingStatus()
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to load billing')
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <NCard title="Billing" :bordered="false">
    <template v-if="billing">
      <NDescriptions bordered :column="1" label-placement="left">
        <NDescriptionsItem label="Plan">
          <NTag type="info">{{ billing.plan }}</NTag>
        </NDescriptionsItem>
        <NDescriptionsItem label="Status">
          <NTag :type="billing.status === 'active' ? 'success' : 'warning'">{{ billing.status }}</NTag>
        </NDescriptionsItem>
        <NDescriptionsItem label="Employees">
          {{ billing.employee_count }} / {{ billing.employee_limit }}
        </NDescriptionsItem>
        <NDescriptionsItem v-if="billing.billing_cycle" label="Billing Cycle">
          {{ billing.billing_cycle }}
        </NDescriptionsItem>
        <NDescriptionsItem v-if="billing.next_billing_date" label="Next Billing">
          {{ billing.next_billing_date }}
        </NDescriptionsItem>
        <NDescriptionsItem v-if="billing.amount" label="Amount">
          ${{ billing.amount }}
        </NDescriptionsItem>
        <NDescriptionsItem label="Features">
          <NTag v-for="f in billing.features" :key="f" size="small" style="margin: 2px">{{ f }}</NTag>
        </NDescriptionsItem>
      </NDescriptions>
    </template>
    <div v-else-if="loading" style="text-align: center; color: #888; padding: 24px">Loading...</div>
  </NCard>
</template>
