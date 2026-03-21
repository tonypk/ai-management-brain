<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { isAuthenticated, logout } from './composables/api'

const router = useRouter()
const route = useRoute()

const showNav = computed(() => route.name !== 'Login' && route.name !== 'Landing' && isAuthenticated())

function handleLogout() {
  logout()
  router.push('/login')
}

const navItems = [
  { path: '/', label: 'Dashboard', icon: '📊' },
  { path: '/employees', label: 'Team', icon: '👥' },
  { path: '/reports', label: 'Reports', icon: '📝' },
  { path: '/mentor', label: 'Mentor', icon: '🧠' },
  { path: '/organization', label: 'Organization', icon: '🏢' },
  { path: '/analytics', label: 'Analytics', icon: '📈' },
]
</script>

<template>
  <div class="app">
    <nav v-if="showNav" class="sidebar">
      <div class="logo">
        <h2>AI Brain</h2>
        <small>Management Dashboard</small>
      </div>
      <div class="nav-items">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ active: route.path === item.path }"
        >
          <span class="icon">{{ item.icon }}</span>
          <span>{{ item.label }}</span>
        </router-link>
      </div>
      <button class="logout-btn" @click="handleLogout">Logout</button>
    </nav>
    <main :class="{ 'with-sidebar': showNav }">
      <router-view />
    </main>
  </div>
</template>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f7fa; color: #1a1a2e; }
.app { display: flex; min-height: 100vh; }
.sidebar { width: 220px; background: #1a1a2e; color: #e0e0e0; display: flex; flex-direction: column; padding: 1.5rem 0; position: fixed; top: 0; left: 0; height: 100vh; }
.logo { padding: 0 1.5rem 1.5rem; border-bottom: 1px solid #2d2d44; }
.logo h2 { color: #fff; font-size: 1.25rem; }
.logo small { color: #888; font-size: 0.75rem; }
.nav-items { flex: 1; padding: 1rem 0; }
.nav-item { display: flex; align-items: center; gap: 0.75rem; padding: 0.75rem 1.5rem; color: #b0b0c0; text-decoration: none; transition: all 0.2s; }
.nav-item:hover { color: #fff; background: rgba(255,255,255,0.05); }
.nav-item.active { color: #fff; background: rgba(99,102,241,0.2); border-right: 3px solid #6366f1; }
.nav-item .icon { font-size: 1.1rem; }
.logout-btn { margin: 1rem 1.5rem; padding: 0.5rem; background: transparent; border: 1px solid #444; color: #888; border-radius: 6px; cursor: pointer; transition: all 0.2s; }
.logout-btn:hover { border-color: #e74c3c; color: #e74c3c; }
main { flex: 1; padding: 2rem; }
main.with-sidebar { margin-left: 220px; }
.card { background: #fff; border-radius: 12px; padding: 1.5rem; box-shadow: 0 1px 3px rgba(0,0,0,0.08); margin-bottom: 1.5rem; }
.card h3 { margin-bottom: 1rem; font-size: 1rem; color: #444; }
.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-bottom: 1.5rem; }
.stat-card { background: #fff; border-radius: 12px; padding: 1.5rem; box-shadow: 0 1px 3px rgba(0,0,0,0.08); text-align: center; }
.stat-card .value { font-size: 2rem; font-weight: 700; color: #6366f1; }
.stat-card .label { color: #888; font-size: 0.85rem; margin-top: 0.25rem; }
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; padding: 0.75rem 1rem; border-bottom: 1px solid #eee; }
th { font-weight: 600; color: #666; font-size: 0.85rem; text-transform: uppercase; letter-spacing: 0.5px; }
.btn { padding: 0.5rem 1rem; border: none; border-radius: 6px; cursor: pointer; font-size: 0.9rem; transition: all 0.2s; }
.btn-primary { background: #6366f1; color: #fff; }
.btn-primary:hover { background: #5558e6; }
.btn-secondary { background: #e5e7eb; color: #374151; }
.btn-secondary:hover { background: #d1d5db; }
input, select { padding: 0.5rem 0.75rem; border: 1px solid #ddd; border-radius: 6px; font-size: 0.9rem; outline: none; transition: border-color 0.2s; }
input:focus, select:focus { border-color: #6366f1; }
.badge { display: inline-block; padding: 0.2rem 0.5rem; border-radius: 4px; font-size: 0.75rem; font-weight: 600; }
.badge-positive { background: #d1fae5; color: #065f46; }
.badge-negative { background: #fee2e2; color: #991b1b; }
.badge-neutral { background: #e0e7ff; color: #3730a3; }
.badge-mixed { background: #fef3c7; color: #92400e; }
.error-msg { color: #e74c3c; font-size: 0.85rem; margin-top: 0.5rem; }
.loading { text-align: center; padding: 2rem; color: #888; }
</style>
