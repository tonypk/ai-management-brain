<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  NCard, NSwitch, NSelect, NCheckboxGroup, NCheckbox,
  NButton, NTag, NSpace, NText, useMessage,
} from 'naive-ui'
import { configureSync, triggerSync } from '@/api/sync'
import type { SyncConfig, ConfigureSyncRequest } from '@/types/sync'

const props = defineProps<{
  storageType: 'notion' | 'sheets'
  config: SyncConfig | null
}>()

const emit = defineEmits<{
  save: [config: SyncConfig]
  'view-logs': []
}>()

const message = useMessage()
const saving = ref(false)
const syncing = ref(false)

const title = computed(() => props.storageType === 'notion' ? 'Notion Sync' : 'Google Sheets Sync')

const isConnected = computed(() => props.config !== null && props.config.is_enabled)

const entityTypes = ref<string[]>(props.config?.entity_types ?? [])
const frequency = ref<number>(props.config?.sync_frequency_minutes ?? 30)

const allEntityTypes = [
  { label: 'Tasks', value: 'tasks' },
  { label: 'Goals', value: 'goals' },
  { label: 'Projects', value: 'projects' },
  { label: 'Metrics', value: 'metrics' },
  { label: 'Reviews', value: 'reviews' },
  { label: 'Training', value: 'training' },
]

const frequencyOptions = [
  { label: '15 min', value: 15 },
  { label: '30 min', value: 30 },
  { label: '1 hour', value: 60 },
  { label: '2 hours', value: 120 },
  { label: '6 hours', value: 360 },
  { label: '12 hours', value: 720 },
  { label: '24 hours', value: 1440 },
]

const lastSyncLabel = computed(() => {
  if (!props.config?.last_sync_at) return null
  const d = new Date(props.config.last_sync_at)
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const diffMin = Math.floor(diffMs / 60000)
  if (diffMin < 1) return 'just now'
  if (diffMin < 60) return `${diffMin} min ago`
  const diffHours = Math.floor(diffMin / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  return d.toLocaleDateString()
})

async function handleToggle(enabled: boolean): Promise<void> {
  saving.value = true
  try {
    const req: ConfigureSyncRequest = {
      storage_type: props.storageType,
      is_enabled: enabled,
      entity_types: entityTypes.value,
      sync_frequency_minutes: frequency.value,
    }
    const updated = await configureSync(req)
    emit('save', updated)
    message.success(enabled ? `${title.value} enabled` : `${title.value} disabled`)
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to update sync config')
  } finally {
    saving.value = false
  }
}

async function handleSave(): Promise<void> {
  saving.value = true
  try {
    const req: ConfigureSyncRequest = {
      storage_type: props.storageType,
      is_enabled: true,
      entity_types: entityTypes.value,
      sync_frequency_minutes: frequency.value,
    }
    const updated = await configureSync(req)
    emit('save', updated)
    message.success('Sync settings saved')
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to save sync config')
  } finally {
    saving.value = false
  }
}

async function handleSyncNow(): Promise<void> {
  if (!props.config) return
  syncing.value = true
  try {
    await triggerSync(props.config.id)
    message.success('Sync triggered')
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to trigger sync')
  } finally {
    syncing.value = false
  }
}

async function handleConnect(): Promise<void> {
  saving.value = true
  try {
    const req: ConfigureSyncRequest = {
      storage_type: props.storageType,
      is_enabled: true,
      entity_types: ['tasks', 'goals'],
      sync_frequency_minutes: 30,
    }
    const updated = await configureSync(req)
    entityTypes.value = updated.entity_types
    frequency.value = updated.sync_frequency_minutes
    emit('save', updated)
    message.success(`${title.value} connected`)
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to connect')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <NCard :title="title" size="small" :bordered="true">
    <!-- Connected state -->
    <template v-if="config">
      <NSpace vertical :size="16">
        <!-- Status row -->
        <NSpace justify="space-between" align="center">
          <NSpace align="center" :size="8">
            <NText>Status:</NText>
            <NTag v-if="isConnected" type="success" size="small">Connected</NTag>
            <NTag v-else type="warning" size="small">Disabled</NTag>
            <NText v-if="lastSyncLabel" depth="3" style="font-size: 12px">
              (last sync: {{ lastSyncLabel }})
            </NText>
          </NSpace>
          <NSwitch :value="config.is_enabled" :loading="saving" @update:value="handleToggle" />
        </NSpace>

        <!-- Settings (only when enabled) -->
        <template v-if="config.is_enabled">
          <!-- Frequency -->
          <NSpace align="center" :size="8">
            <NText>Frequency:</NText>
            <NSelect
              v-model:value="frequency"
              :options="frequencyOptions"
              size="small"
              style="width: 120px"
            />
          </NSpace>

          <!-- Entity types -->
          <NSpace vertical :size="4">
            <NText>Sync entities:</NText>
            <NCheckboxGroup v-model:value="entityTypes">
              <NSpace :size="12">
                <NCheckbox
                  v-for="et in allEntityTypes"
                  :key="et.value"
                  :value="et.value"
                  :label="et.label"
                />
              </NSpace>
            </NCheckboxGroup>
          </NSpace>

          <!-- Last sync status -->
          <NSpace v-if="config.last_sync_status" align="center" :size="8">
            <NText>Last status:</NText>
            <NTag
              :type="config.last_sync_status === 'success' ? 'success' : config.last_sync_status === 'partial' ? 'warning' : 'error'"
              size="small"
            >
              {{ config.last_sync_status }}
            </NTag>
          </NSpace>

          <!-- Actions -->
          <NSpace :size="8">
            <NButton type="primary" size="small" :loading="saving" @click="handleSave">
              Save
            </NButton>
            <NButton size="small" :loading="syncing" @click="handleSyncNow">
              Sync Now
            </NButton>
            <NButton size="small" quaternary @click="emit('view-logs')">
              View Logs
            </NButton>
          </NSpace>
        </template>
      </NSpace>
    </template>

    <!-- Not connected state -->
    <template v-else>
      <NSpace justify="space-between" align="center">
        <NSpace align="center" :size="8">
          <NText>Status:</NText>
          <NTag type="default" size="small">Not connected</NTag>
        </NSpace>
        <NButton type="primary" size="small" :loading="saving" @click="handleConnect">
          Connect {{ storageType === 'notion' ? 'Notion' : 'Sheets' }}
        </NButton>
      </NSpace>
    </template>
  </NCard>
</template>
