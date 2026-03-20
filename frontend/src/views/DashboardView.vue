<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getDashboard, type DashboardStats } from '../composables/api'

const stats = ref<DashboardStats | null>(null)
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  try {
    const res = await getDashboard()
    stats.value = res.data
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <h2>Dashboard</h2>
    <p v-if="loading" class="loading">Loading...</p>
    <p v-else-if="error" class="error-msg">{{ error }}</p>
    <template v-else-if="stats">
      <div class="stats-grid" style="margin-top: 1.5rem">
        <div class="stat-card">
          <div class="value">{{ stats.employee_count }}</div>
          <div class="label">Team Members</div>
        </div>
        <div class="stat-card">
          <div class="value">{{ stats.today_submissions }}</div>
          <div class="label">Today's Reports</div>
        </div>
        <div class="stat-card">
          <div class="value" style="font-size: 1.5rem; text-transform: capitalize">{{ stats.current_mentor }}</div>
          <div class="label">Active Mentor</div>
        </div>
        <div class="stat-card">
          <div class="value" style="font-size: 1.25rem">{{ stats.last_summary_date || 'No summaries yet' }}</div>
          <div class="label">Last Summary</div>
        </div>
      </div>

      <div class="card">
        <h3>Quick Actions</h3>
        <div style="display: flex; gap: 0.75rem; flex-wrap: wrap">
          <router-link to="/employees" class="btn btn-primary">Manage Team</router-link>
          <router-link to="/reports" class="btn btn-secondary">View Reports</router-link>
          <router-link to="/mentor" class="btn btn-secondary">Configure Mentor</router-link>
        </div>
      </div>
    </template>
  </div>
</template>
