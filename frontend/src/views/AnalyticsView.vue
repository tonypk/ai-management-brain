<template>
  <div class="analytics">
    <h1>Team Analytics</h1>

    <!-- Health Score -->
    <div class="health-card" v-if="overview">
      <div class="health-score" :class="healthClass">{{ overview.health_score }}</div>
      <div class="health-label">Team Health Score</div>
    </div>

    <!-- Today's Stats -->
    <div class="stats-row" v-if="overview">
      <div class="stat-card">
        <div class="stat-value">{{ overview.today.reports }}/{{ overview.today.employees }}</div>
        <div class="stat-label">Submitted Today</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{ (overview.today.submission_rate * 100).toFixed(0) }}%</div>
        <div class="stat-label">Submission Rate</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{ topSentiment }}</div>
        <div class="stat-label">Top Sentiment</div>
      </div>
    </div>

    <!-- 7-Day Trend -->
    <div class="card" v-if="overview">
      <h2>7-Day Submission Trend</h2>
      <div class="trend-chart">
        <div v-for="day in overview.trend_7d" :key="day.date" class="trend-bar-wrapper">
          <div class="trend-bar" :style="{ height: (day.rate * 100) + '%' }"></div>
          <div class="trend-label">{{ day.date.slice(5) }}</div>
          <div class="trend-count">{{ day.count }}</div>
        </div>
      </div>
    </div>

    <!-- Sentiment Distribution -->
    <div class="card" v-if="overview">
      <h2>Sentiment Distribution</h2>
      <div class="sentiment-bars">
        <div v-for="(count, sentiment) in overview.sentiment" :key="sentiment" class="sentiment-row">
          <span class="sentiment-label">{{ sentiment }}</span>
          <div class="sentiment-bar-bg">
            <div class="sentiment-bar-fill" :class="'sentiment-' + sentiment" :style="{ width: sentimentPercent(count) + '%' }"></div>
          </div>
          <span class="sentiment-count">{{ count }}</span>
        </div>
      </div>
    </div>

    <!-- Employee Activity -->
    <div class="card">
      <h2>Employee Activity (Last 7 Days)</h2>
      <table v-if="activity.length">
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
      <p v-else class="empty">No employee data available.</p>
    </div>

    <p v-if="error" class="error">{{ error }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getAnalyticsOverview, getEmployeeActivity } from '../composables/api'

const overview = ref<any>(null)
const activity = ref<any[]>([])
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
    const [ov, act] = await Promise.all([
      getAnalyticsOverview(),
      getEmployeeActivity(),
    ])
    overview.value = ov.data
    activity.value = act.data
  } catch (e: any) {
    error.value = e.message
  }
})
</script>

<style scoped>
.analytics { max-width: 900px; }
.analytics h1 { margin-bottom: 24px; }

.health-card {
  text-align: center;
  padding: 32px;
  background: #f8fafc;
  border-radius: 12px;
  margin-bottom: 24px;
}
.health-score {
  font-size: 72px;
  font-weight: 800;
  line-height: 1;
}
.health-label { color: #64748b; margin-top: 8px; }
.health-good { color: #22c55e; }
.health-ok { color: #f59e0b; }
.health-bad { color: #ef4444; }

.stats-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}
.stat-card {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 20px;
  text-align: center;
}
.stat-value { font-size: 28px; font-weight: 700; }
.stat-label { color: #64748b; font-size: 14px; margin-top: 4px; }

.card {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 24px;
  margin-bottom: 24px;
}
.card h2 { font-size: 18px; margin-bottom: 16px; }

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
  background: #2563eb;
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

table { width: 100%; border-collapse: collapse; }
th, td { padding: 10px 12px; text-align: left; border-bottom: 1px solid #f1f5f9; }
th { font-size: 13px; color: #64748b; }
.text-red { color: #ef4444; font-weight: 600; }
.badge {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
}
.badge-positive { background: #dcfce7; color: #166534; }
.badge-neutral { background: #f1f5f9; color: #475569; }
.badge-negative { background: #fee2e2; color: #991b1b; }
.badge-stressed { background: #fef3c7; color: #92400e; }
.error { color: #ef4444; }
.empty { color: #94a3b8; text-align: center; padding: 24px; }
</style>
