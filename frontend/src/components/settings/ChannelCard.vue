<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NCard, NSpace, NSwitch, NTag, NButton, NGrid, NGi, useMessage } from 'naive-ui'
import { getChannelConfig, updateChannelConfig, testChannel } from '@/api/settings'
import type { ChannelConfig } from '@/types'

const message = useMessage()
const loading = ref(false)
const config = ref<ChannelConfig | null>(null)

const channels = ['telegram', 'slack', 'lark', 'signal'] as const

onMounted(async () => {
  loading.value = true
  try {
    config.value = await getChannelConfig()
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to load channels')
  } finally {
    loading.value = false
  }
})

function isEnabled(ch: string): boolean {
  return config.value?.enabled_channels?.includes(ch) ?? false
}

function isConfigured(ch: string): boolean {
  return config.value?.channels?.find(c => c.type === ch)?.configured ?? false
}

async function toggleChannel(ch: string): Promise<void> {
  if (!config.value) return
  const enabled = [...config.value.enabled_channels]
  const idx = enabled.indexOf(ch)
  if (idx >= 0) {
    enabled.splice(idx, 1)
  } else {
    enabled.push(ch)
  }
  try {
    await updateChannelConfig({ enabled_channels: enabled })
    config.value = { ...config.value, enabled_channels: enabled }
    message.success(`${ch} ${idx >= 0 ? 'disabled' : 'enabled'}`)
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to update')
  }
}

async function handleTest(ch: string): Promise<void> {
  try {
    const sent = await testChannel(ch, 'test')
    if (sent) {
      message.success(`Test message sent via ${ch}`)
    } else {
      message.warning('Test message was not delivered')
    }
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Test failed')
  }
}
</script>

<template>
  <NCard title="Channels" :bordered="false">
    <NGrid :cols="2" :x-gap="16" :y-gap="16" responsive="screen" :item-responsive="true">
      <NGi v-for="ch in channels" :key="ch" span="2 m:1">
        <NCard size="small" :bordered="true">
          <NSpace justify="space-between" align="center">
            <NSpace align="center" :size="12">
              <span style="font-weight: 600; text-transform: capitalize">{{ ch }}</span>
              <NTag v-if="isConfigured(ch)" type="success" size="small">Configured</NTag>
              <NTag v-else type="warning" size="small">Not configured</NTag>
            </NSpace>
            <NSpace :size="8">
              <NSwitch :value="isEnabled(ch)" @update:value="toggleChannel(ch)" />
              <NButton size="small" :disabled="!isEnabled(ch)" @click="handleTest(ch)">Test</NButton>
            </NSpace>
          </NSpace>
        </NCard>
      </NGi>
    </NGrid>
  </NCard>
</template>
