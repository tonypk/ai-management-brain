<script setup lang="ts">
import { ref } from 'vue'
import { listReports, getSummary, type Report, type Summary } from '../composables/api'

const today = new Date().toISOString().slice(0, 10)
const selectedDate = ref(today)
const reports = ref<Report[]>([])
const summary = ref<Summary | null>(null)
const loading = ref(false)
const error = ref('')

async function loadReports() {
  loading.value = true
  error.value = ''
  summary.value = null
  reports.value = []

  try {
    const [reportsRes, summaryRes] = await Promise.allSettled([
      listReports(selectedDate.value),
      getSummary(selectedDate.value),
    ])

    if (reportsRes.status === 'fulfilled') {
      reports.value = reportsRes.value.data
    }
    if (summaryRes.status === 'fulfilled') {
      summary.value = summaryRes.value.data
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function sentimentClass(s: string | undefined): string {
  if (!s) return 'badge-neutral'
  if (s.includes('positive')) return 'badge-positive'
  if (s.includes('negative')) return 'badge-negative'
  if (s.includes('mixed')) return 'badge-mixed'
  return 'badge-neutral'
}
</script>

<template>
  <div>
    <h2>Daily Reports</h2>

    <div class="card" style="margin-top: 1.5rem">
      <form @submit.prevent="loadReports" style="display: flex; gap: 0.75rem; align-items: center">
        <input v-model="selectedDate" type="date" />
        <button type="submit" class="btn btn-primary" :disabled="loading">
          {{ loading ? 'Loading...' : 'Load Reports' }}
        </button>
      </form>
    </div>

    <p v-if="error" class="error-msg">{{ error }}</p>

    <div v-if="summary" class="card">
      <h3>Daily Summary - {{ summary.summary_date }}</h3>
      <div class="stats-grid" style="margin-bottom: 1rem">
        <div class="stat-card">
          <div class="value">{{ Math.round(summary.submission_rate * 100) }}%</div>
          <div class="label">Submission Rate</div>
        </div>
        <div class="stat-card">
          <div class="value">{{ summary.blockers_count }}</div>
          <div class="label">Blockers</div>
        </div>
      </div>
      <pre style="white-space: pre-wrap; font-size: 0.9rem; color: #444; line-height: 1.6">{{ summary.content }}</pre>
    </div>

    <div v-if="reports.length > 0" class="card">
      <h3>Individual Reports ({{ reports.length }})</h3>
      <div v-for="r in reports" :key="r.id" style="border-bottom: 1px solid #eee; padding: 1rem 0">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem">
          <strong>{{ r.employee_name }}</strong>
          <div style="display: flex; gap: 0.5rem">
            <span v-if="r.sentiment" :class="'badge ' + sentimentClass(r.sentiment)">
              {{ r.sentiment }}
            </span>
            <span style="color: #888; font-size: 0.8rem">{{ new Date(r.submitted_at).toLocaleTimeString() }}</span>
          </div>
        </div>
        <div v-if="r.answers" style="font-size: 0.9rem; color: #555">
          <div v-for="(answer, question) in r.answers" :key="String(question)" style="margin-bottom: 0.5rem">
            <div style="color: #888; font-size: 0.8rem">{{ question }}</div>
            <div>{{ answer }}</div>
          </div>
        </div>
        <div v-if="r.blockers" style="margin-top: 0.5rem; padding: 0.5rem; background: #fef3c7; border-radius: 6px; font-size: 0.85rem">
          <strong>Blockers:</strong> {{ r.blockers }}
        </div>
      </div>
    </div>

    <div v-if="!loading && reports.length === 0 && !summary" class="card">
      <p style="text-align: center; color: #888; padding: 1rem">
        Select a date and click "Load Reports" to view daily submissions.
      </p>
    </div>
  </div>
</template>
