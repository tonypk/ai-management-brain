<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getDashboard, getAnalyticsOverview, getEmployeeActivity, type DashboardStats, type AnalyticsOverview, type EmployeeActivity } from '../composables/api'

const stats = ref<DashboardStats | null>(null)
const overview = ref<AnalyticsOverview | null>(null)
const activity = ref<EmployeeActivity[]>([])
const loading = ref(true)
const error = ref('')

const healthClass = computed(() => {
  if (!overview.value) return ''
  const s = overview.value.health_score
  if (s >= 80) return 'health-good'
  if (s >= 50) return 'health-ok'
  return 'health-bad'
})

const topSentiment = computed(() => {
  if (!overview.value) return '—'
  const s = overview.value.sentiment
  let max = 0, top = 'neutral'
  for (const [k, v] of Object.entries(s)) {
    if ((v as number) > max) { max = v as number; top = k }
  }
  return max > 0 ? top : '—'
})

const totalSentiment = computed(() => {
  if (!overview.value) return 0
  return Object.values(overview.value.sentiment).reduce((a: number, b: any) => a + b, 0)
})

function sentimentPercent(count: number): number {
  const total = totalSentiment.value as number
  return total > 0 ? (count / total) * 100 : 0
}

onMounted(async () => {
  try {
    const [dash, ov, act] = await Promise.all([
      getDashboard(),
      getAnalyticsOverview(),
      getEmployeeActivity(),
    ])
    stats.value = dash.data
    overview.value = ov.data
    activity.value = act.data
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="dashboard">
    <h2>Dashboard</h2>
    <p v-if="loading" class="loading">Loading...</p>
    <p v-else-if="error" class="error-msg">{{ error }}</p>
    <template v-else>

      <!-- Health Score -->
      <div class="health-card" v-if="overview">
        <div class="health-score" :class="healthClass">{{ overview.health_score }}</div>
        <div class="health-label">Team Health Score</div>
      </div>

      <!-- Stats Cards -->
      <div class="stats-row">
        <div class="stat-card" v-if="overview">
          <div class="stat-value">{{ overview.today.reports }}/{{ overview.today.employees }}</div>
          <div class="stat-label">Submitted Today</div>
        </div>
        <div class="stat-card" v-if="overview">
          <div class="stat-value">{{ (overview.today.submission_rate * 100).toFixed(0) }}%</div>
          <div class="stat-label">Submission Rate</div>
        </div>
        <div class="stat-card" v-if="overview">
          <div class="stat-value">{{ topSentiment }}</div>
          <div class="stat-label">Top Sentiment</div>
        </div>
        <div class="stat-card" v-if="stats">
          <div class="stat-value" style="font-size: 1.25rem; text-transform: capitalize">{{ stats.current_mentor }}</div>
          <div class="stat-label">Active Mentor</div>
        </div>
      </div>

      <!-- Quick Actions -->
      <div class="card">
        <h3>Quick Actions</h3>
        <div style="display: flex; gap: 0.75rem; flex-wrap: wrap">
          <router-link to="/employees" class="btn btn-primary">Manage Team</router-link>
          <router-link to="/reports" class="btn btn-secondary">View Reports</router-link>
          <router-link to="/mentor" class="btn btn-secondary">Configure Mentor</router-link>
        </div>
      </div>

      <!-- 7-Day Trend -->
      <div class="card" v-if="overview && overview.trend_7d.length">
        <h3>7-Day Submission Trend</h3>
        <div class="trend-chart">
          <div v-for="day in overview.trend_7d" :key="day.date" class="trend-bar-wrapper">
            <div class="trend-bar" :style="{ height: (day.rate * 100) + '%' }"></div>
            <div class="trend-label">{{ day.date.slice(5) }}</div>
            <div class="trend-count">{{ day.count }}</div>
          </div>
        </div>
      </div>

      <!-- Sentiment Distribution -->
      <div class="card" v-if="overview && Object.keys(overview.sentiment).length">
        <h3>Sentiment Distribution</h3>
        <div class="sentiment-bars">
          <div v-for="(count, sentiment) in overview.sentiment" :key="sentiment" class="sentiment-row">
            <span class="sentiment-label">{{ sentiment }}</span>
            <div class="sentiment-bar-bg">
              <div class="sentiment-bar-fill" :class="'sentiment-' + sentiment" :style="{ width: sentimentPercent(count as number) + '%' }"></div>
            </div>
            <span class="sentiment-count">{{ count }}</span>
          </div>
        </div>
      </div>

      <!-- Employee Activity -->
      <div class="card" v-if="activity.length">
        <h3>Employee Activity (Last 7 Days)</h3>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Submitted</th>
              <th>Missed</th>
              <th>Sentiment</th>
              <th>Culture</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="emp in activity" :key="emp.id">
              <td>{{ emp.name }}</td>
              <td>{{ emp.submitted_7d }}/7</td>
              <td>
                <span :class="{ 'text-red': emp.missed_7d >= 3 }">{{ emp.missed_7d }}</span>
              </td>
              <td>
                <span class="badge" :class="'badge-' + emp.last_sentiment">{{ emp.last_sentiment || '—' }}</span>
              </td>
              <td>{{ emp.culture_code }}</td>
            </tr>
          </tbody>
        </table>
      </div>

    </template>
  </div>
</template>

<style scoped>
.dashboard { max-width: 900px; }

.health-card {
  text-align: center;
  padding: 2rem;
  background: #f8fafc;
  border-radius: 12px;
  margin: 1.5rem 0;
}
.health-score {
  font-size: 4rem;
  font-weight: 800;
  line-height: 1;
}
.health-label { color: #64748b; margin-top: 0.5rem; }
.health-good { color: #22c55e; }
.health-ok { color: #f59e0b; }
.health-bad { color: #ef4444; }

.stats-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.trend-chart {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  height: 120px;
}
.trend-bar-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
  justify-content: flex-end;
}
.trend-bar {
  width: 100%;
  max-width: 48px;
  background: #6366f1;
  border-radius: 4px 4px 0 0;
  min-height: 4px;
  transition: height 0.3s;
}
.trend-label { font-size: 12px; color: #94a3b8; margin-top: 4px; }
.trend-count { font-size: 12px; font-weight: 600; }

.sentiment-bars { display: flex; flex-direction: column; gap: 12px; }
.sentiment-row { display: flex; align-items: center; gap: 12px; }
.sentiment-label { width: 80px; font-size: 14px; text-transform: capitalize; }
.sentiment-bar-bg {
  flex: 1;
  height: 20px;
  background: #f1f5f9;
  border-radius: 10px;
  overflow: hidden;
}
.sentiment-bar-fill {
  height: 100%;
  border-radius: 10px;
  transition: width 0.3s;
}
.sentiment-positive { background: #22c55e; }
.sentiment-neutral { background: #94a3b8; }
.sentiment-negative { background: #ef4444; }
.sentiment-stressed { background: #f59e0b; }
.sentiment-count { width: 30px; text-align: right; font-weight: 600; }

.text-red { color: #ef4444; font-weight: 600; }
</style>
