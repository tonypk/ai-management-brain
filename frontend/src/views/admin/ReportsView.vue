<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  listAdminReports,
  getReportStats,
  type AdminReport,
} from '../../composables/api'

const reports = ref<AdminReport[]>([])
const loading = ref(true)
const error = ref('')

// Pagination
const page = ref(1)
const limit = ref(20)
const total = ref(0)
const hasMore = ref(false)

// Filters
const dateFrom = ref('')
const dateTo = ref('')
const filterEmployee = ref('')
const filterChannel = ref('')

// Stats
const channelStats = ref<{ channel: string; count: number }[]>([])

// Expanded row
const expandedId = ref<string | null>(null)

function sentimentClass(s: string | undefined): string {
  if (!s) return 'badge-neutral'
  if (s.includes('positive')) return 'badge-positive'
  if (s.includes('negative')) return 'badge-negative'
  if (s.includes('mixed')) return 'badge-mixed'
  return 'badge-neutral'
}

async function loadReports() {
  loading.value = true
  error.value = ''
  try {
    const [reportsRes, statsRes] = await Promise.allSettled([
      listAdminReports({
        page: page.value,
        limit: limit.value,
        date_from: dateFrom.value || undefined,
        date_to: dateTo.value || undefined,
        employee_id: filterEmployee.value || undefined,
        channel: filterChannel.value || undefined,
      }),
      getReportStats(dateFrom.value || undefined, dateTo.value || undefined),
    ])

    if (reportsRes.status === 'fulfilled') {
      reports.value = reportsRes.value.data
      total.value = reportsRes.value.meta.total
      hasMore.value = reportsRes.value.meta.has_more
    }
    if (statsRes.status === 'fulfilled') {
      channelStats.value = statsRes.value.data
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function applyFilters() {
  page.value = 1
  loadReports()
}

function prevPage() {
  if (page.value > 1) {
    page.value--
    loadReports()
  }
}

function nextPage() {
  if (hasMore.value) {
    page.value++
    loadReports()
  }
}

function toggleExpand(id: string) {
  expandedId.value = expandedId.value === id ? null : id
}

onMounted(loadReports)
</script>

<template>
  <div>
    <h2>Admin Reports</h2>

    <!-- Filters -->
    <div class="card" style="margin-top: 1.5rem">
      <form @submit.prevent="applyFilters" style="display: flex; gap: 0.75rem; align-items: flex-end; flex-wrap: wrap">
        <div>
          <label class="field-label">From</label>
          <input v-model="dateFrom" type="date" />
        </div>
        <div>
          <label class="field-label">To</label>
          <input v-model="dateTo" type="date" />
        </div>
        <div>
          <label class="field-label">Employee ID</label>
          <input v-model="filterEmployee" placeholder="Employee ID" />
        </div>
        <div>
          <label class="field-label">Channel</label>
          <select v-model="filterChannel">
            <option value="">All</option>
            <option value="telegram">Telegram</option>
            <option value="signal">Signal</option>
            <option value="slack">Slack</option>
            <option value="lark">Lark</option>
          </select>
        </div>
        <button type="submit" class="btn btn-primary" :disabled="loading">
          {{ loading ? 'Loading...' : 'Filter' }}
        </button>
      </form>
    </div>

    <!-- Stats -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="value">{{ total }}</div>
        <div class="label">Total Reports</div>
      </div>
      <div v-for="cs in channelStats" :key="cs.channel" class="stat-card">
        <div class="value">{{ cs.count }}</div>
        <div class="label" style="text-transform: capitalize">{{ cs.channel }}</div>
      </div>
    </div>

    <p v-if="error" class="error-msg">{{ error }}</p>

    <!-- Table -->
    <div v-if="!loading" class="card">
      <table>
        <thead>
          <tr>
            <th>Employee</th>
            <th>Date</th>
            <th>Channel</th>
            <th>Time</th>
            <th>Sentiment</th>
            <th>Blockers</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="reports.length === 0">
            <td colspan="6" style="text-align: center; color: #888; padding: 2rem">
              No reports found.
            </td>
          </tr>
          <template v-for="r in reports" :key="r.id">
            <tr class="report-row" @click="toggleExpand(r.id)">
              <td><strong>{{ r.employee_name }}</strong></td>
              <td>{{ r.report_date }}</td>
              <td>
                <span class="badge badge-neutral" style="text-transform: capitalize">{{ r.channel }}</span>
              </td>
              <td>{{ new Date(r.submitted_at).toLocaleTimeString() }}</td>
              <td>
                <span v-if="r.sentiment" :class="'badge ' + sentimentClass(r.sentiment)">
                  {{ r.sentiment }}
                </span>
                <span v-else style="color: #888">-</span>
              </td>
              <td>{{ r.blockers || '-' }}</td>
            </tr>
            <tr v-if="expandedId === r.id" class="expanded-row">
              <td colspan="6">
                <div class="expanded-content">
                  <h4 style="margin-bottom: 0.5rem">Answers</h4>
                  <div v-for="(answer, question) in r.answers" :key="String(question)" style="margin-bottom: 0.5rem">
                    <div style="color: #888; font-size: 0.8rem">{{ question }}</div>
                    <div style="font-size: 0.9rem">{{ answer }}</div>
                  </div>
                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>

      <!-- Pagination -->
      <div class="pagination">
        <button class="btn btn-secondary" :disabled="page <= 1" @click="prevPage">Previous</button>
        <span class="page-info">Page {{ page }} ({{ total }} total)</span>
        <button class="btn btn-secondary" :disabled="!hasMore" @click="nextPage">Next</button>
      </div>
    </div>

    <p v-if="loading" class="loading">Loading...</p>
  </div>
</template>

<style scoped>
.field-label {
  display: block;
  font-size: 0.85rem;
  color: #666;
  margin-bottom: 0.25rem;
}
.report-row {
  cursor: pointer;
  transition: background 0.2s;
}
.report-row:hover {
  background: rgba(99, 102, 241, 0.03);
}
.expanded-row td {
  background: #f9fafb;
  border-bottom: 2px solid #e5e7eb;
}
.expanded-content {
  padding: 0.75rem;
}
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding: 1rem 0 0.5rem;
}
.page-info {
  color: #666;
  font-size: 0.9rem;
}
</style>
