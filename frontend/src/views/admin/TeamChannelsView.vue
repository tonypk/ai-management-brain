<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  listEmployeesWithChannels,
  updateEmployeeChannels,
  updateEmployeePreferred,
  type EmployeeWithChannels,
} from '../../composables/api'

const employees = ref<EmployeeWithChannels[]>([])
const loading = ref(true)
const error = ref('')
const search = ref('')
const savingId = ref('')

// Inline editing
const editingCell = ref<{ id: string; field: string } | null>(null)
const editValue = ref('')

const filteredEmployees = computed(() => {
  if (!search.value) return employees.value
  const q = search.value.toLowerCase()
  return employees.value.filter(
    (e) => e.name.toLowerCase().includes(q) || e.role.toLowerCase().includes(q),
  )
})

async function loadEmployees() {
  try {
    const res = await listEmployeesWithChannels()
    employees.value = res.data
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function startEdit(emp: EmployeeWithChannels, field: string) {
  editingCell.value = { id: emp.id, field }
  editValue.value = (emp as any)[field] || ''
}

function isEditing(empId: string, field: string): boolean {
  return editingCell.value?.id === empId && editingCell.value?.field === field
}

async function saveEdit(emp: EmployeeWithChannels) {
  if (!editingCell.value) return
  const { field } = editingCell.value
  const newVal = editValue.value.trim()

  // Skip if value unchanged
  if (newVal === ((emp as any)[field] || '')) {
    editingCell.value = null
    return
  }

  savingId.value = emp.id
  error.value = ''
  try {
    await updateEmployeeChannels(emp.id, { [field]: newVal })
    // Update local state immutably
    employees.value = employees.value.map((e) =>
      e.id === emp.id ? { ...e, [field]: newVal } : e,
    )
  } catch (e: any) {
    error.value = e.message
  } finally {
    savingId.value = ''
    editingCell.value = null
  }
}

async function handlePreferredChange(emp: EmployeeWithChannels, channel: string) {
  savingId.value = emp.id
  error.value = ''
  try {
    await updateEmployeePreferred(emp.id, channel)
    employees.value = employees.value.map((e) =>
      e.id === emp.id ? { ...e, preferred_channel: channel } : e,
    )
  } catch (e: any) {
    error.value = e.message
  } finally {
    savingId.value = ''
  }
}

onMounted(loadEmployees)
</script>

<template>
  <div>
    <h2>Team Channel Assignments</h2>

    <div class="card" style="margin-top: 1.5rem">
      <input
        v-model="search"
        placeholder="Search employees..."
        style="width: 100%; max-width: 300px"
      />
    </div>

    <p v-if="loading" class="loading">Loading...</p>
    <p v-if="error" class="error-msg">{{ error }}</p>

    <div v-if="!loading" class="card">
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Role</th>
            <th>Telegram</th>
            <th>Signal</th>
            <th>Slack</th>
            <th>Lark</th>
            <th>Preferred</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="filteredEmployees.length === 0">
            <td colspan="7" style="text-align: center; color: #888; padding: 2rem">
              No employees found.
            </td>
          </tr>
          <tr v-for="emp in filteredEmployees" :key="emp.id" :class="{ 'row-saving': savingId === emp.id }">
            <td><strong>{{ emp.name }}</strong></td>
            <td>{{ emp.role }}</td>
            <td>
              <span :class="emp.telegram_id ? 'badge badge-positive' : 'badge badge-neutral'">
                {{ emp.telegram_id ? 'Connected' : '-' }}
              </span>
            </td>
            <td class="editable-cell" @click="startEdit(emp, 'signal_phone')">
              <input
                v-if="isEditing(emp.id, 'signal_phone')"
                v-model="editValue"
                @blur="saveEdit(emp)"
                @keyup.enter="saveEdit(emp)"
                class="inline-input"
                autofocus
              />
              <span v-else>{{ emp.signal_phone || '-' }}</span>
            </td>
            <td class="editable-cell" @click="startEdit(emp, 'slack_id')">
              <input
                v-if="isEditing(emp.id, 'slack_id')"
                v-model="editValue"
                @blur="saveEdit(emp)"
                @keyup.enter="saveEdit(emp)"
                class="inline-input"
                autofocus
              />
              <span v-else>{{ emp.slack_id || '-' }}</span>
            </td>
            <td class="editable-cell" @click="startEdit(emp, 'lark_id')">
              <input
                v-if="isEditing(emp.id, 'lark_id')"
                v-model="editValue"
                @blur="saveEdit(emp)"
                @keyup.enter="saveEdit(emp)"
                class="inline-input"
                autofocus
              />
              <span v-else>{{ emp.lark_id || '-' }}</span>
            </td>
            <td>
              <select
                :value="emp.preferred_channel"
                @change="handlePreferredChange(emp, ($event.target as HTMLSelectElement).value)"
                class="inline-select"
              >
                <option value="telegram">Telegram</option>
                <option value="signal">Signal</option>
                <option value="slack">Slack</option>
                <option value="lark">Lark</option>
              </select>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.editable-cell {
  cursor: pointer;
  min-width: 100px;
}
.editable-cell:hover {
  background: rgba(99, 102, 241, 0.05);
}
.inline-input {
  width: 100%;
  padding: 0.25rem 0.5rem;
  border: 1px solid #6366f1;
  border-radius: 4px;
  font-size: 0.9rem;
  outline: none;
}
.inline-select {
  padding: 0.25rem 0.5rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 0.85rem;
  background: #fff;
  cursor: pointer;
}
.row-saving {
  opacity: 0.6;
}
</style>
