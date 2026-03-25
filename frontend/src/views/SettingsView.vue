<script setup lang="ts">
import { NTabs, NTabPane } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import TenantForm from '@/components/settings/TenantForm.vue'
import ChannelCard from '@/components/settings/ChannelCard.vue'
import SchedulerTable from '@/components/settings/SchedulerTable.vue'
import ApiKeyManager from '@/components/settings/ApiKeyManager.vue'
import BillingStatus from '@/components/settings/BillingStatus.vue'
</script>

<template>
  <div>
    <PageHeader title="Settings" :breadcrumbs="[{ label: 'Dashboard', to: '/' }, { label: 'Settings' }]" />

    <NTabs type="line" animated>
      <NTabPane name="tenant" tab="Organization">
        <TenantForm />
      </NTabPane>
      <NTabPane name="channels" tab="Channels">
        <ChannelCard />
      </NTabPane>
      <NTabPane name="scheduler" tab="Scheduler">
        <SchedulerTable />
      </NTabPane>
      <NTabPane name="api-keys" tab="API Keys">
        <ApiKeyManager />
      </NTabPane>
      <NTabPane name="billing" tab="Billing">
        <BillingStatus />
      </NTabPane>
    </NTabs>
  </div>
</template>
