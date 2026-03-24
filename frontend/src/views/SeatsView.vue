<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  listSeats,
  createSeat,
  updateSeat,
  deleteSeat,
  boardDiscuss,
  listMentorsWithDomain,
  type Seat,
  type MentorWithDomain,
  type BoardResponse,
} from '../composables/api'

const seats = ref<Seat[]>([])
const mentors = ref<MentorWithDomain[]>([])
const loading = ref(true)
const error = ref('')
const savingId = ref('')

// Create modal
const showCreate = ref(false)
const newSeatType = ref('')
const newPersonaId = ref('')
const newScope = ref('')

// Edit modal
const editingSeat = ref<Seat | null>(null)
const editTitle = ref('')
const editPersonaId = ref('')
const editScope = ref('')

// Board discussion
const showBoard = ref(false)
const boardTopic = ref('')
const boardLoading = ref(false)
const boardResponses = ref<BoardResponse[]>([])
const boardSynthesis = ref('')

const predefinedTypes = [
  { value: 'ceo', label: 'CEO — Chief Executive Officer' },
  { value: 'cfo', label: 'CFO — Chief Financial Officer' },
  { value: 'cmo', label: 'CMO — Chief Marketing Officer' },
  { value: 'cto', label: 'CTO — Chief Technology Officer' },
  { value: 'chro', label: 'CHRO — Chief Human Resources Officer' },
  { value: 'coo', label: 'COO — Chief Operations Officer' },
]

const availableTypes = computed(() => {
  const used = new Set(seats.value.map((s) => s.seat_type))
  return predefinedTypes.filter((t) => !used.has(t.value))
})

const filteredMentors = computed(() => {
  return mentors.value
})

async function load() {
  try {
    const [seatsRes, mentorsRes] = await Promise.all([listSeats(), listMentorsWithDomain()])
    seats.value = seatsRes.data
    mentors.value = mentorsRes.data
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!newSeatType.value || !newPersonaId.value) return
  error.value = ''
  try {
    const res = await createSeat({
      seat_type: newSeatType.value,
      persona_id: newPersonaId.value,
      scope: newScope.value,
    })
    seats.value = [...seats.value, res.data]
    showCreate.value = false
    newSeatType.value = ''
    newPersonaId.value = ''
    newScope.value = ''
  } catch (e: any) {
    error.value = e.message
  }
}

function startEdit(seat: Seat) {
  editingSeat.value = seat
  editTitle.value = seat.title
  editPersonaId.value = seat.persona_id
  editScope.value = seat.scope
}

async function saveEdit() {
  if (!editingSeat.value) return
  const id = editingSeat.value.id
  savingId.value = id
  error.value = ''
  try {
    const res = await updateSeat(id, {
      title: editTitle.value,
      persona_id: editPersonaId.value,
      scope: editScope.value,
    })
    seats.value = seats.value.map((s) => (s.id === id ? { ...s, ...res.data } : s))
    editingSeat.value = null
  } catch (e: any) {
    error.value = e.message
  } finally {
    savingId.value = ''
  }
}

async function handleDelete(seat: Seat) {
  if (!confirm(`Remove ${seat.title} (${seat.seat_type})?`)) return
  savingId.value = seat.id
  error.value = ''
  try {
    await deleteSeat(seat.id)
    seats.value = seats.value.filter((s) => s.id !== seat.id)
  } catch (e: any) {
    error.value = e.message
  } finally {
    savingId.value = ''
  }
}

async function runBoardDiscussion() {
  if (!boardTopic.value.trim()) return
  boardLoading.value = true
  boardResponses.value = []
  boardSynthesis.value = ''
  error.value = ''
  try {
    const res = await boardDiscuss(boardTopic.value)
    boardResponses.value = res.data.responses
    boardSynthesis.value = res.data.synthesis
  } catch (e: any) {
    error.value = e.message
  } finally {
    boardLoading.value = false
  }
}

function getMentorName(personaId: string): string {
  const m = mentors.value.find((m) => m.id === personaId)
  return m ? `${m.name} (${m.name_en})` : personaId
}

function getDomainBadge(personaId: string): string {
  const m = mentors.value.find((m) => m.id === personaId)
  return m?.domain || ''
}

onMounted(load)
</script>

<template>
  <div>
    <h2>C-Suite Team</h2>
    <p style="color: #888; margin-top: 0.25rem; font-size: 0.9rem">
      Assemble your AI management team. Assign expert personas to organizational seats.
    </p>

    <div style="display: flex; gap: 0.75rem; margin-top: 1.5rem">
      <button class="btn btn-primary" @click="showCreate = true" :disabled="availableTypes.length === 0">
        + Add Seat
      </button>
      <button class="btn btn-secondary" @click="showBoard = true" :disabled="seats.length === 0">
        Board Discussion
      </button>
    </div>

    <p v-if="loading" class="loading">Loading...</p>
    <p v-if="error" class="error-msg">{{ error }}</p>

    <div v-if="!loading && seats.length === 0" class="card" style="margin-top: 1.5rem">
      <p style="text-align: center; color: #888; padding: 2rem">
        No seats assigned yet. Click "+ Add Seat" to build your C-Suite team.
      </p>
    </div>

    <!-- Seat Cards -->
    <div v-if="!loading && seats.length > 0" class="seats-grid">
      <div v-for="seat in seats" :key="seat.id" class="card seat-card" :class="{ 'row-saving': savingId === seat.id }">
        <div class="seat-header">
          <span class="seat-type">{{ seat.seat_type.toUpperCase() }}</span>
          <span :class="seat.is_active ? 'badge badge-positive' : 'badge badge-negative'">
            {{ seat.is_active ? 'Active' : 'Inactive' }}
          </span>
        </div>
        <h3 style="margin: 0.5rem 0 0.25rem">{{ seat.title }}</h3>
        <p class="persona-name">{{ getMentorName(seat.persona_id) }}</p>
        <span v-if="getDomainBadge(seat.persona_id)" class="badge badge-neutral" style="margin-top: 0.25rem">
          {{ getDomainBadge(seat.persona_id) }}
        </span>
        <p v-if="seat.scope" style="color: #888; font-size: 0.8rem; margin-top: 0.5rem">{{ seat.scope }}</p>
        <div style="margin-top: 1rem; display: flex; gap: 0.5rem">
          <button class="btn btn-secondary btn-sm" @click="startEdit(seat)">Edit</button>
          <button class="btn btn-danger btn-sm" @click="handleDelete(seat)">Remove</button>
        </div>
      </div>
    </div>

    <!-- Create Modal -->
    <div v-if="showCreate" class="modal-overlay" @click.self="showCreate = false">
      <div class="modal">
        <h3>Add Seat</h3>
        <div class="form-group">
          <label>Seat Type</label>
          <select v-model="newSeatType">
            <option value="" disabled>Select type...</option>
            <option v-for="t in availableTypes" :key="t.value" :value="t.value">{{ t.label }}</option>
          </select>
        </div>
        <div class="form-group">
          <label>Persona</label>
          <select v-model="newPersonaId">
            <option value="" disabled>Select persona...</option>
            <option v-for="m in filteredMentors" :key="m.id" :value="m.id">
              {{ m.name }} ({{ m.name_en }}) — {{ m.domain }}
            </option>
          </select>
        </div>
        <div class="form-group">
          <label>Scope (optional)</label>
          <input v-model="newScope" placeholder="Responsibilities description..." />
        </div>
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="showCreate = false">Cancel</button>
          <button class="btn btn-primary" @click="handleCreate" :disabled="!newSeatType || !newPersonaId">Create</button>
        </div>
      </div>
    </div>

    <!-- Edit Modal -->
    <div v-if="editingSeat" class="modal-overlay" @click.self="editingSeat = null">
      <div class="modal">
        <h3>Edit {{ editingSeat.seat_type.toUpperCase() }}</h3>
        <div class="form-group">
          <label>Title</label>
          <input v-model="editTitle" />
        </div>
        <div class="form-group">
          <label>Persona</label>
          <select v-model="editPersonaId">
            <option v-for="m in filteredMentors" :key="m.id" :value="m.id">
              {{ m.name }} ({{ m.name_en }}) — {{ m.domain }}
            </option>
          </select>
        </div>
        <div class="form-group">
          <label>Scope</label>
          <input v-model="editScope" />
        </div>
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="editingSeat = null">Cancel</button>
          <button class="btn btn-primary" @click="saveEdit">Save</button>
        </div>
      </div>
    </div>

    <!-- Board Discussion Modal -->
    <div v-if="showBoard" class="modal-overlay" @click.self="!boardLoading && (showBoard = false)">
      <div class="modal" style="max-width: 640px">
        <h3>Board Discussion</h3>
        <div v-if="!boardResponses.length" class="form-group">
          <label>Topic</label>
          <input v-model="boardTopic" placeholder="Enter a strategic question..." @keyup.enter="runBoardDiscussion" />
        </div>
        <div v-if="boardLoading" class="loading">Discussing... (this may take a minute)</div>
        <div v-if="boardResponses.length > 0" class="board-results">
          <div v-for="r in boardResponses" :key="r.seat_type" class="board-response">
            <strong>{{ r.title }} ({{ r.seat_type.toUpperCase() }})</strong>
            <p>{{ r.content }}</p>
          </div>
          <div class="board-synthesis">
            <strong>Synthesis</strong>
            <p>{{ boardSynthesis }}</p>
          </div>
        </div>
        <div class="modal-actions">
          <button v-if="boardResponses.length" class="btn btn-secondary" @click="boardResponses = []; boardSynthesis = ''; boardTopic = ''">New Topic</button>
          <button class="btn btn-secondary" @click="showBoard = false; boardResponses = []; boardSynthesis = ''; boardTopic = ''">Close</button>
          <button v-if="!boardResponses.length" class="btn btn-primary" @click="runBoardDiscussion" :disabled="!boardTopic.trim() || boardLoading">Discuss</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.seats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
  margin-top: 1.5rem;
}
.seat-card {
  position: relative;
}
.seat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.seat-type {
  font-size: 0.75rem;
  font-weight: 700;
  color: #6366f1;
  letter-spacing: 1px;
}
.persona-name {
  color: #555;
  font-size: 0.9rem;
  margin: 0;
}
.btn-sm {
  padding: 0.3rem 0.75rem;
  font-size: 0.8rem;
}
.btn-danger {
  background: #fee2e2;
  color: #991b1b;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}
.btn-danger:hover {
  background: #fecaca;
}
.row-saving {
  opacity: 0.6;
}
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}
.modal {
  background: #fff;
  border-radius: 12px;
  padding: 2rem;
  width: 100%;
  max-width: 420px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  max-height: 80vh;
  overflow-y: auto;
}
.modal h3 { margin-bottom: 1.5rem; }
.form-group { margin-bottom: 1rem; }
.form-group label {
  display: block;
  font-size: 0.85rem;
  color: #666;
  margin-bottom: 0.25rem;
}
.form-group input:not([type]),
.form-group select { width: 100%; }
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1.5rem;
}
.board-results {
  max-height: 50vh;
  overflow-y: auto;
}
.board-response {
  margin-bottom: 1rem;
  padding: 0.75rem;
  background: #f9fafb;
  border-radius: 8px;
}
.board-response p {
  margin: 0.5rem 0 0;
  font-size: 0.9rem;
  color: #333;
  white-space: pre-wrap;
}
.board-synthesis {
  margin-top: 1rem;
  padding: 0.75rem;
  background: #eef2ff;
  border-radius: 8px;
  border-left: 3px solid #6366f1;
}
.board-synthesis p {
  margin: 0.5rem 0 0;
  font-size: 0.9rem;
  color: #333;
  white-space: pre-wrap;
}
</style>
