<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  listAIRoles,
  listSuggestions,
  approveSuggestion,
  rejectSuggestion,
  type AIRoleInstance,
  type AISuggestion,
} from '../composables/api'

const roles = ref<AIRoleInstance[]>([])
const suggestions = ref<AISuggestion[]>([])
const loading = ref(true)
const error = ref('')
const expandedId = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [rolesRes, suggestionsRes] = await Promise.all([
      listAIRoles(),
      listSuggestions(),
    ])
    roles.value = rolesRes.data
    suggestions.value = suggestionsRes.data
  } catch (e: any) {
    error.value = e.message || 'Failed to load AI roles'
  } finally {
    loading.value = false
  }
}

async function handleApprove(id: string) {
  try {
    await approveSuggestion(id)
    await load()
  } catch (e: any) {
    error.value = e.message || 'Failed to approve'
  }
}

async function handleReject(id: string) {
  try {
    await rejectSuggestion(id)
    await load()
  } catch (e: any) {
    error.value = e.message || 'Failed to reject'
  }
}

function toggleExpand(id: string) {
  expandedId.value = expandedId.value === id ? null : id
}

function formatDate(d: string) {
  return new Date(d).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

onMounted(load)
</script>

<template>
  <div>
    <h1>AI Roles</h1>
    <p class="subtitle">AI-powered C-level roles assisting your organization</p>

    <p v-if="error" class="error-msg">{{ error }}</p>
    <p v-if="loading" class="loading">Loading...</p>

    <template v-if="!loading">
      <!-- Active Roles -->
      <div class="card">
        <h3>Active Roles</h3>
        <div v-if="roles.length === 0" class="empty">
          No AI roles activated. Activate an organization plan with AI support roles to get started.
        </div>
        <div v-else class="roles-grid">
          <div v-for="role in roles" :key="role.id" class="role-card">
            <div class="role-header">
              <span class="role-icon">&#129302;</span>
              <div>
                <div class="role-title">{{ role.title }}</div>
                <div class="role-id">{{ role.role_id }}</div>
              </div>
            </div>
            <div class="role-meta">
              <span>Mentor: <strong>{{ role.mentor_id }}</strong></span>
              <span v-if="role.pending_count > 0" class="badge badge-mixed">
                {{ role.pending_count }} pending
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Suggestions Queue -->
      <div class="card">
        <h3>Suggestions Queue</h3>
        <div v-if="suggestions.length === 0" class="empty">
          No pending suggestions from AI roles.
        </div>
        <table v-else>
          <thead>
            <tr>
              <th>Role</th>
              <th>Capability</th>
              <th>Title</th>
              <th>Created</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="s in suggestions" :key="s.id">
              <tr class="suggestion-row" @click="toggleExpand(s.id)">
                <td>{{ s.role_title }}</td>
                <td><span class="badge badge-neutral">{{ s.capability }}</span></td>
                <td>{{ s.title }}</td>
                <td>{{ formatDate(s.created_at) }}</td>
                <td class="actions" @click.stop>
                  <button class="btn btn-primary btn-sm" @click="handleApprove(s.id)">Approve</button>
                  <button class="btn btn-secondary btn-sm" @click="handleReject(s.id)">Reject</button>
                </td>
              </tr>
              <tr v-if="expandedId === s.id" class="expand-row">
                <td colspan="5">
                  <div class="suggestion-content">{{ s.content }}</div>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
    </template>
  </div>
</template>

<style scoped>
h1 { margin-bottom: 0.25rem; }
.subtitle { color: #888; margin-bottom: 1.5rem; font-size: 0.9rem; }
.empty { color: #888; padding: 1rem 0; }
.roles-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1rem; }
.role-card { background: #f9fafb; border: 1px solid #e5e7eb; border-radius: 8px; padding: 1rem; }
.role-header { display: flex; align-items: center; gap: 0.75rem; margin-bottom: 0.75rem; }
.role-icon { font-size: 1.5rem; }
.role-title { font-weight: 600; font-size: 1rem; }
.role-id { color: #888; font-size: 0.8rem; }
.role-meta { display: flex; justify-content: space-between; align-items: center; font-size: 0.85rem; color: #666; }
.suggestion-row { cursor: pointer; }
.suggestion-row:hover { background: #f9fafb; }
.expand-row td { padding: 0 1rem 1rem; }
.suggestion-content { background: #f3f4f6; padding: 1rem; border-radius: 8px; white-space: pre-wrap; font-size: 0.9rem; line-height: 1.5; }
.actions { display: flex; gap: 0.5rem; }
.btn-sm { padding: 0.25rem 0.5rem; font-size: 0.8rem; }
</style>
