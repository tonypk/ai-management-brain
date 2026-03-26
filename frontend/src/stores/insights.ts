import { defineStore } from 'pinia'
import { computed } from 'vue'
import { useLocalStorage } from '@/composables'
import type { InsightRecord, InsightsStorage, DigestRecord, DigestsStorage, DigestPeriod } from '@/types'

const INSIGHTS_KEY = 'brain_insights'
const DIGESTS_KEY = 'brain_digests'

function now(): string {
  return new Date().toISOString()
}

function uid(): string {
  return crypto.randomUUID()
}

function defaultInsightsStorage(): InsightsStorage {
  return { meta: { version: 1, updated_at: now() }, insights: [] }
}

function defaultDigestsStorage(): DigestsStorage {
  return { meta: { version: 1, updated_at: now() }, digests: [] }
}

export const useInsightsStore = defineStore('insights', () => {
  const insightsData = useLocalStorage<InsightsStorage>(INSIGHTS_KEY, defaultInsightsStorage())
  const digestsData = useLocalStorage<DigestsStorage>(DIGESTS_KEY, defaultDigestsStorage())

  // ── Insights ──
  const insights = computed(() => insightsData.value.insights)

  function addInsight(content: string, contextSummary: string, healthScore: number, alertCount: number): InsightRecord {
    const record: InsightRecord = {
      id: uid(),
      content,
      context_summary: contextSummary,
      health_score: healthScore,
      alert_count: alertCount,
      created_at: now(),
    }
    insightsData.value = {
      meta: { version: 1, updated_at: now() },
      insights: [record, ...insightsData.value.insights],
    }
    return record
  }

  function deleteInsight(id: string): void {
    insightsData.value = {
      meta: { version: 1, updated_at: now() },
      insights: insightsData.value.insights.filter((r) => r.id !== id),
    }
  }

  // ── Digests ──
  const digests = computed(() => digestsData.value.digests)

  function addDigest(period: DigestPeriod, periodLabel: string, content: string): DigestRecord {
    const record: DigestRecord = {
      id: uid(),
      period,
      period_label: periodLabel,
      content,
      created_at: now(),
    }
    digestsData.value = {
      meta: { version: 1, updated_at: now() },
      digests: [record, ...digestsData.value.digests],
    }
    return record
  }

  function deleteDigest(id: string): void {
    digestsData.value = {
      meta: { version: 1, updated_at: now() },
      digests: digestsData.value.digests.filter((r) => r.id !== id),
    }
  }

  return {
    insights,
    addInsight,
    deleteInsight,
    digests,
    addDigest,
    deleteDigest,
  }
})
