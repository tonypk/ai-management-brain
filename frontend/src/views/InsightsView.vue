<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NButton, NIcon, NSpin, useMessage } from 'naive-ui'
import { SparklesOutline } from '@vicons/ionicons5'
import PageHeader from '@/components/shared/PageHeader.vue'
import DataSummaryCards from '@/components/insights/DataSummaryCards.vue'
import InsightPanel from '@/components/insights/InsightPanel.vue'
import InsightHistory from '@/components/insights/InsightHistory.vue'
import { useDataCollector } from '@/composables'
import { useInsightsStore } from '@/stores/insights'
import { chatWithSeat } from '@/api/seats'
import type { CollectedData } from '@/composables/useDataCollector'

const message = useMessage()
const store = useInsightsStore()
const { collect, formatContext, summarize } = useDataCollector()

const loading = ref(false)
const dataLoading = ref(true)
const collectedData = ref<CollectedData | null>(null)

const latestInsight = computed(() => store.insights[0] ?? null)
const pastInsights = computed(() => store.insights.slice(1))

const healthScore = computed(() => collectedData.value?.analytics?.health_score ?? 0)
const alertCount = computed(() => collectedData.value?.alerts.length ?? 0)
const submissionRate = computed(() => {
  const rate = collectedData.value?.analytics?.today?.submission_rate
  return rate ? Math.round(rate * 100) : 0
})
const atRiskCount = computed(() =>
  collectedData.value?.alerts.filter((a) => a.severity === 'critical').length ?? 0,
)

onMounted(async () => {
  try {
    collectedData.value = await collect()
  } catch {
    // Cards will show zeros
  } finally {
    dataLoading.value = false
  }
})

async function generateInsight() {
  if (!collectedData.value) {
    collectedData.value = await collect()
  }

  loading.value = true
  try {
    const context = formatContext(collectedData.value)
    const prompt = `You are the CEO advisor analyzing the current state of this organization.
Based on the data below, provide a concise management insight report.

Sections:
## Team Health — Overall assessment
## Key Risks — Top 3 risks that need attention
## Opportunities — Positive trends to leverage
## Recommendations — Top 5 actionable items

Keep each section to 2-4 bullet points. Be specific with names and numbers.

${context}`

    const resp = await chatWithSeat('ceo', prompt)
    const summary = summarize(collectedData.value)
    store.addInsight(resp.content, summary, healthScore.value, alertCount.value)
    message.success('Insight generated')
  } catch (err: unknown) {
    message.error(`Failed to generate insight: ${err instanceof Error ? err.message : 'Unknown error'}`)
  } finally {
    loading.value = false
  }
}

function handleDelete(id: string) {
  store.deleteInsight(id)
}
</script>

<template>
  <div>
    <PageHeader title="AI Insights">
      <template #actions>
        <NButton type="primary" :loading="loading" :disabled="dataLoading" @click="generateInsight">
          <template #icon><NIcon :component="SparklesOutline" /></template>
          Generate Insights
        </NButton>
      </template>
    </PageHeader>

    <NSpin :show="dataLoading">
      <DataSummaryCards
        :health-score="healthScore"
        :alert-count="alertCount"
        :submission-rate="submissionRate"
        :at-risk-count="atRiskCount"
      />
    </NSpin>

    <InsightPanel
      :insight="latestInsight"
      :loading="loading"
      @regenerate="generateInsight"
    />

    <InsightHistory
      :records="pastInsights"
      @delete="handleDelete"
    />
  </div>
</template>
