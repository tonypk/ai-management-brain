<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NButton, NInput, NSpace, NAlert, useMessage, useDialog, type DataTableColumn } from 'naive-ui'
import { listApiKeys, createApiKey, revokeApiKey } from '@/api/auth'
import type { ApiKey } from '@/types'

const message = useMessage()
const dialog = useDialog()
const loading = ref(false)
const keys = ref<ApiKey[]>([])
const newKeyName = ref('')
const creating = ref(false)
const createdKey = ref('')

onMounted(loadKeys)

async function loadKeys(): Promise<void> {
  loading.value = true
  try {
    keys.value = await listApiKeys()
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to load API keys')
  } finally {
    loading.value = false
  }
}

async function handleCreate(): Promise<void> {
  if (!newKeyName.value.trim()) return
  creating.value = true
  try {
    const result = await createApiKey(newKeyName.value.trim())
    createdKey.value = result.key
    newKeyName.value = ''
    message.success('API key created — copy it now, it won\'t be shown again!')
    await loadKeys()
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to create')
  } finally {
    creating.value = false
  }
}

function handleRevoke(id: string, name: string): void {
  dialog.warning({
    title: 'Revoke API Key',
    content: `Are you sure you want to revoke "${name}"?`,
    positiveText: 'Revoke',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await revokeApiKey(id)
        message.success('API key revoked')
        await loadKeys()
      } catch (e: unknown) {
        message.error(e instanceof Error ? e.message : 'Failed to revoke')
      }
    },
  })
}

const columns: DataTableColumn<ApiKey>[] = [
  { title: 'Name', key: 'name' },
  { title: 'Prefix', key: 'prefix' },
  { title: 'Created', key: 'created_at' },
  { title: 'Last Used', key: 'last_used_at', render(row) { return row.last_used_at || 'Never' } },
  {
    title: 'Action',
    key: 'action',
    render(row) {
      return h(NButton, { size: 'small', type: 'error', ghost: true, onClick: () => handleRevoke(row.id, row.name) }, () => 'Revoke')
    },
  },
]
</script>

<template>
  <NCard title="API Keys" :bordered="false">
    <NAlert v-if="createdKey" type="success" closable style="margin-bottom: 16px" @close="createdKey = ''">
      <div style="font-weight: 600; margin-bottom: 4px">New API Key (copy now!):</div>
      <code style="background: #f0f0f0; padding: 4px 8px; border-radius: 4px; word-break: break-all">{{ createdKey }}</code>
    </NAlert>

    <NSpace style="margin-bottom: 16px" :size="8">
      <NInput v-model:value="newKeyName" placeholder="Key name" style="width: 200px" @keydown.enter="handleCreate" />
      <NButton type="primary" :loading="creating" @click="handleCreate">Create</NButton>
    </NSpace>

    <NDataTable :columns="columns" :data="keys" :loading="loading" :bordered="false" size="small" />
  </NCard>
</template>
