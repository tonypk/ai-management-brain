<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  listAdminMemories,
  searchAdminMemories,
  deleteAdminMemory,
  getMemoryStats,
  type MemoryItem,
  type MemoryStats,
} from '../../composables/api'

const memories = ref<MemoryItem[]>([])
const stats = ref<MemoryStats | null>(null)
const loading = ref(true)
const error = ref('')
const success = ref('')

// Pagination
const page = ref(1)
const limit = ref(20)
const total = ref(0)
const hasMore = ref(false)

// Filters
const filterType = ref('')
const filterTier = ref('')
const filterEmployee = ref('')

// Search
const searchQuery = ref('')
const searching = ref(false)
const isSearchMode = ref(false)

// Detail modal
const selectedMemory = ref<MemoryItem | null>(null)

// Delete confirmation
const deletingId = ref('')

const memoryTypes = ['observation', 'insight', 'pattern', 'fact']
const memoryTiers = ['short_term', 'long_term', 'core']

async function loadMemories() {
  loading.value = true
  error.value = ''
  isSearchMode.value = false
  try {
    const [memRes, statsRes] = await Promise.allSettled([
      listAdminMemories({
        page: page.value,
        limit: limit.value,
        type: filterType.value || undefined,
        tier: filterTier.value || undefined,
        employee_id: filterEmployee.value || undefined,
      }),
      getMemoryStats(),
    ])

    if (memRes.status === 'fulfilled') {
      memories.value = memRes.value.data
      total.value = memRes.value.meta.total
      hasMore.value = memRes.value.meta.has_more
    }
    if (statsRes.status === 'fulfilled') {
      stats.value = statsRes.value.data
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function handleSearch() {
  if (!searchQuery.value.trim()) {
    loadMemories()
    return
  }
  searching.value = true
  error.value = ''
  isSearchMode.value = true
  try {
    const res = await searchAdminMemories(searchQuery.value.trim())
    memories.value = res.data
    total.value = res.data.length
    hasMore.value = false
  } catch (e: any) {
    error.value = e.message
  } finally {
    searching.value = false
  }
}

function applyFilters() {
  page.value = 1
  searchQuery.value = ''
  loadMemories()
}

async function handleDelete(id: string) {
  if (!confirm('Are you sure you want to delete this memory?')) return
  deletingId.value = id
  error.value = ''
  success.value = ''
  try {
    await deleteAdminMemory(id)
    success.value = 'Memory deleted successfully.'
    memories.value = memories.value.filter((m) => m.id !== id)
    total.value = Math.max(0, total.value - 1)
    if (selectedMemory.value?.id === id) {
      selectedMemory.value = null
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    deletingId.value = ''
  }
}

function truncate(text: string, max: number): string {
  if (text.length <= max) return text
  return text.slice(0, max) + '...'
}

function prevPage() {
  if (page.value > 1) {
    page.value--
    loadMemories()
  }
}

function nextPage() {
  if (hasMore.value) {
    page.value++
    loadMemories()
  }
}

onMounted(loadMemories)
</script>

<template>
  <div>
    <h2>Memory Management</h2>

    <!-- Stats -->
    <div class="stats-grid" style="margin-top: 1.5rem">
      <div class="stat-card">
        <div class="value">{{ stats?.total ?? '-' }}</div>
        <div class="label">Total Memories</div>
      </div>
    </div>

    <!-- Filters -->
    <div class="card">
      <div style="display: flex; gap: 0.75rem; align-items: flex-end; flex-wrap: wrap; margin-bottom: 1rem">
        <div>
          <label class="field-label">Type</label>
          <select v-model="filterType" @change="applyFilters">
            <option value="">All Types</option>
            <option v-for="t in memoryTypes" :key="t" :value="t" style="text-transform: capitalize">{{ t }}</option>
          </select>
        </div>
        <div>
          <label class="field-label">Tier</label>
          <select v-model="filterTier" @change="applyFilters">
            <option value="">All Tiers</option>
            <option v-for="t in memoryTiers" :key="t" :value="t">{{ t.replace('_', ' ') }}</option>
          </select>
        </div>
        <div>
          <label class="field-label">Employee ID</label>
          <input v-model="filterEmployee" placeholder="Employee ID" @keyup.enter="applyFilters" />
        </div>
        <button class="btn btn-secondary" @click="applyFilters">Filter</button>
      </div>

      <!-- Semantic Search -->
      <form @submit.prevent="handleSearch" style="display: flex; gap: 0.75rem; align-items: center">
        <input
          v-model="searchQuery"
          placeholder="Semantic search..."
          style="flex: 1; max-width: 400px"
        />
        <button type="submit" class="btn btn-primary" :disabled="searching">
          {{ searching ? 'Searching...' : 'Search' }}
        </button>
        <button v-if="isSearchMode" type="button" class="btn btn-secondary" @click="searchQuery = ''; loadMemories()">
          Clear Search
        </button>
      </form>
    </div>

    <p v-if="error" class="error-msg">{{ error }}</p>
    <p v-if="success" style="color: #065f46; font-size: 0.85rem; margin-bottom: 0.5rem">{{ success }}</p>

    <!-- Table -->
    <p v-if="loading" class="loading">Loading...</p>
    <div v-else class="card">
      <table>
        <thead>
          <tr>
            <th>Content</th>
            <th>Type</th>
            <th>Tier</th>
            <th>Importance</th>
            <th>Employee</th>
            <th>Created</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="memories.length === 0">
            <td colspan="7" style="text-align: center; color: #888; padding: 2rem">
              No memories found.
            </td>
          </tr>
          <tr
            v-for="m in memories"
            :key="m.id"
            class="memory-row"
            @click="selectedMemory = m"
          >
            <td style="max-width: 300px">{{ truncate(m.content, 100) }}</td>
            <td>
              <span class="badge badge-neutral" style="text-transform: capitalize">{{ m.memory_type }}</span>
            </td>
            <td>
              <span class="badge" :class="m.memory_tier === 'core' ? 'badge-positive' : m.memory_tier === 'long_term' ? 'badge-mixed' : 'badge-neutral'">
                {{ m.memory_tier.replace('_', ' ') }}
              </span>
            </td>
            <td>{{ m.importance }}</td>
            <td>{{ m.employee_id || '-' }}</td>
            <td style="font-size: 0.85rem; color: #666">{{ new Date(m.created_at).toLocaleDateString() }}</td>
            <td>
              <button
                class="btn btn-sm"
                style="background: #fee2e2; color: #991b1b"
                :disabled="deletingId === m.id"
                @click.stop="handleDelete(m.id)"
              >
                {{ deletingId === m.id ? '...' : 'Delete' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="!isSearchMode" class="pagination">
        <button class="btn btn-secondary" :disabled="page <= 1" @click="prevPage">Previous</button>
        <span class="page-info">Page {{ page }} ({{ total }} total)</span>
        <button class="btn btn-secondary" :disabled="!hasMore" @click="nextPage">Next</button>
      </div>
    </div>

    <!-- Detail Modal -->
    <div v-if="selectedMemory" class="modal-overlay" @click.self="selectedMemory = null">
      <div class="modal-content card">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem">
          <h3>Memory Detail</h3>
          <button class="btn btn-secondary btn-sm" @click="selectedMemory = null">Close</button>
        </div>

        <div class="detail-field">
          <label class="field-label">Content</label>
          <pre class="detail-pre">{{ selectedMemory.content }}</pre>
        </div>

        <div v-if="selectedMemory.summary" class="detail-field">
          <label class="field-label">Summary</label>
          <p>{{ selectedMemory.summary }}</p>
        </div>

        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-top: 1rem">
          <div>
            <label class="field-label">Type</label>
            <span class="badge badge-neutral" style="text-transform: capitalize">{{ selectedMemory.memory_type }}</span>
          </div>
          <div>
            <label class="field-label">Tier</label>
            <span>{{ selectedMemory.memory_tier }}</span>
          </div>
          <div>
            <label class="field-label">Importance</label>
            <span>{{ selectedMemory.importance }}</span>
          </div>
          <div>
            <label class="field-label">Access Count</label>
            <span>{{ selectedMemory.access_count }}</span>
          </div>
          <div>
            <label class="field-label">Employee ID</label>
            <span>{{ selectedMemory.employee_id || '-' }}</span>
          </div>
          <div>
            <label class="field-label">Created</label>
            <span>{{ new Date(selectedMemory.created_at).toLocaleString() }}</span>
          </div>
          <div>
            <label class="field-label">Updated</label>
            <span>{{ new Date(selectedMemory.updated_at).toLocaleString() }}</span>
          </div>
          <div>
            <label class="field-label">Expires</label>
            <span>{{ selectedMemory.expires_at ? new Date(selectedMemory.expires_at).toLocaleString() : 'Never' }}</span>
          </div>
        </div>

        <div v-if="selectedMemory.metadata && Object.keys(selectedMemory.metadata).length > 0" class="detail-field" style="margin-top: 1rem">
          <label class="field-label">Metadata</label>
          <pre class="detail-pre">{{ JSON.stringify(selectedMemory.metadata, null, 2) }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.field-label {
  display: block;
  font-size: 0.85rem;
  color: #666;
  margin-bottom: 0.25rem;
}
.memory-row {
  cursor: pointer;
  transition: background 0.2s;
}
.memory-row:hover {
  background: rgba(99, 102, 241, 0.03);
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
.btn-sm {
  padding: 0.35rem 0.75rem;
  font-size: 0.8rem;
}
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-content {
  width: 600px;
  max-width: 90vw;
  max-height: 80vh;
  overflow-y: auto;
}
.detail-field {
  margin-top: 0.75rem;
}
.detail-pre {
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 0.85rem;
  color: #444;
  background: #f9fafb;
  padding: 0.75rem;
  border-radius: 6px;
  line-height: 1.5;
  max-height: 200px;
  overflow-y: auto;
}
</style>
