<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NTabs, NTabPane, NSpace, useMessage } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import TenantForm from '@/components/settings/TenantForm.vue'
import ChannelCard from '@/components/settings/ChannelCard.vue'
import SchedulerTable from '@/components/settings/SchedulerTable.vue'
import ApiKeyManager from '@/components/settings/ApiKeyManager.vue'
import BillingStatusComp from '@/components/settings/BillingStatus.vue'
import SyncConfigPanel from '@/components/settings/SyncConfigPanel.vue'
import SyncHistoryTable from '@/components/settings/SyncHistoryTable.vue'
import { listSyncConfigs, listSyncLogs } from '@/api/sync'
import type { SyncConfig, SyncLog } from '@/types/sync'

const message = useMessage()

const notionConfig = ref<SyncConfig | null>(null)
const sheetsConfig = ref<SyncConfig | null>(null)
const syncLogs = ref<SyncLog[]>([])
const logsLoading = ref(false)
const showLogs = ref(false)
const activeLogConfigId = ref<string | null>(null)

onMounted(async () => {
  try {
    const configs = await listSyncConfigs()
    notionConfig.value = configs.find(c => c.storage_type === 'notion') ?? null
    sheetsConfig.value = configs.find(c => c.storage_type === 'sheets') ?? null
  } catch {
    // Sync may not be set up yet — silently ignore
  }
})

function handleSaveNotion(updated: SyncConfig): void {
  notionConfig.value = updated
}

function handleSaveSheets(updated: SyncConfig): void {
  sheetsConfig.value = updated
}

async function handleViewLogs(configId: string | null): Promise<void> {
  if (!configId) return
  activeLogConfigId.value = configId
  showLogs.value = true
  logsLoading.value = true
  try {
    syncLogs.value = await listSyncLogs(configId)
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to load sync logs')
    syncLogs.value = []
  } finally {
    logsLoading.value = false
  }
}
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
      <NTabPane name="sync" tab="Sync">
        <NSpace vertical :size="16">
          <SyncConfigPanel
            storage-type="notion"
            :config="notionConfig"
            @save="handleSaveNotion"
            @view-logs="handleViewLogs(notionConfig?.id ?? null)"
          />
          <SyncConfigPanel
            storage-type="sheets"
            :config="sheetsConfig"
            @save="handleSaveSheets"
            @view-logs="handleViewLogs(sheetsConfig?.id ?? null)"
          />
          <SyncHistoryTable
            v-if="showLogs"
            :logs="syncLogs"
            :loading="logsLoading"
          />
        </NSpace>
      </NTabPane>
      <NTabPane name="billing" tab="Billing">
        <BillingStatusComp />
      </NTabPane>
    </NTabs>
  </div>
</template>
