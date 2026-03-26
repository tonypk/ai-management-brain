<script setup lang="ts">
import { NCard, NDescriptions, NDescriptionsItem, NTag } from 'naive-ui'
import type { EmployeeWithChannels } from '@/types'

defineProps<{
  channels: EmployeeWithChannels | null
}>()

const channelLabel: Record<string, string> = {
  telegram: 'Telegram',
  slack: 'Slack',
  lark: 'Lark',
  signal: 'Signal',
}
</script>

<template>
  <NCard title="Channels" :bordered="false" size="small" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <template v-if="channels">
      <NDescriptions label-placement="left" :column="1" size="small">
        <NDescriptionsItem v-if="channels.telegram_id" label="Telegram">
          Connected
        </NDescriptionsItem>
        <NDescriptionsItem v-if="channels.slack_id" label="Slack">
          {{ channels.slack_id }}
        </NDescriptionsItem>
        <NDescriptionsItem v-if="channels.lark_id" label="Lark">
          {{ channels.lark_id }}
        </NDescriptionsItem>
        <NDescriptionsItem v-if="channels.signal_phone" label="Signal">
          {{ channels.signal_phone }}
        </NDescriptionsItem>
        <NDescriptionsItem label="Preferred">
          <NTag type="info" size="small" round>
            {{ channelLabel[channels.preferred_channel] || channels.preferred_channel || 'None' }}
          </NTag>
        </NDescriptionsItem>
      </NDescriptions>
    </template>
    <div v-else style="color: #888; font-size: 13px">No channel information available</div>
  </NCard>
</template>
