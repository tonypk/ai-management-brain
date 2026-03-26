<script setup lang="ts">
import { ref, computed } from 'vue'
import { NButton, NIcon, useMessage } from 'naive-ui'
import { NewspaperOutline } from '@vicons/ionicons5'
import PageHeader from '@/components/shared/PageHeader.vue'
import DigestPeriodSelector from '@/components/digest/DigestPeriodSelector.vue'
import DigestContent from '@/components/digest/DigestContent.vue'
import DigestHistory from '@/components/digest/DigestHistory.vue'
import { useDataCollector } from '@/composables'
import { useInsightsStore } from '@/stores/insights'
import { chatWithSeat } from '@/api/seats'
import type { DigestPeriod } from '@/types'

const message = useMessage()
const store = useInsightsStore()
const { collect, formatContext } = useDataCollector()

const loading = ref(false)
const period = ref<DigestPeriod>('weekly')

function defaultLabel(): string {
  const now = new Date()
  const monday = new Date(now)
  monday.setDate(monday.getDate() - ((monday.getDay() + 6) % 7))
  return `Week of ${monday.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}`
}

const selectedLabel = ref(defaultLabel())

const latestDigest = computed(() => store.digests[0] ?? null)
const pastDigests = computed(() => store.digests.slice(1))

async function generateDigest() {
  loading.value = true
  try {
    const data = await collect()
    const context = formatContext(data)
    const periodWord = period.value === 'weekly' ? 'weekly' : 'monthly'
    const nextPeriod = period.value === 'weekly' ? 'Week' : 'Month'

    const prompt = `You are the CEO advisor writing a ${periodWord} management digest.
Based on the data below, create an executive summary.

Sections:
## Executive Summary — 2-3 sentence overview
## Key Metrics — Important numbers and trends
## Highlights — What went well
## Concerns — What needs attention
## Action Items for Next ${nextPeriod} — Top 5 priorities

${context}`

    const resp = await chatWithSeat('ceo', prompt)
    store.addDigest(period.value, selectedLabel.value, resp.content)
    message.success('Digest generated')
  } catch (err: unknown) {
    message.error(`Failed to generate digest: ${err instanceof Error ? err.message : 'Unknown error'}`)
  } finally {
    loading.value = false
  }
}

function handleDelete(id: string) {
  store.deleteDigest(id)
}
</script>

<template>
  <div>
    <PageHeader title="Weekly Digest">
      <template #actions>
        <NButton type="primary" :loading="loading" @click="generateDigest">
          <template #icon><NIcon :component="NewspaperOutline" /></template>
          Generate Digest
        </NButton>
      </template>
    </PageHeader>

    <DigestPeriodSelector
      v-model:period="period"
      v-model:selected-label="selectedLabel"
    />

    <DigestContent
      :digest="latestDigest"
      :loading="loading"
      @regenerate="generateDigest"
    />

    <DigestHistory
      :records="pastDigests"
      @delete="handleDelete"
    />
  </div>
</template>
