<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NTabs, NTabPane, NSpin } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import AlertList from '@/components/alerts/AlertList.vue'
import AlertRulesDisplay from '@/components/alerts/AlertRulesDisplay.vue'
import { getAlerts } from '@/api/alerts'
import type { Alert } from '@/types'

const loading = ref(true)
const alerts = ref<Alert[]>([])

onMounted(async () => {
  try {
    alerts.value = await getAlerts()
  } catch {
    // show empty state
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <PageHeader title="Alert Center" :breadcrumbs="[{ label: 'Dashboard', to: '/' }, { label: 'Alerts' }]" />

    <NSpin :show="loading">
      <NTabs type="line" animated>
        <NTabPane name="active" tab="Active Alerts">
          <AlertList :alerts="alerts" />
        </NTabPane>
        <NTabPane name="rules" tab="Alert Rules">
          <AlertRulesDisplay />
        </NTabPane>
      </NTabs>
    </NSpin>
  </div>
</template>
