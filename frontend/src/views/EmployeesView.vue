<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listEmployees, createEmployee, type Employee } from '../composables/api'

const employees = ref<Employee[]>([])
const loading = ref(true)
const error = ref('')

const showAdd = ref(false)
const newName = ref('')
const newCulture = ref('default')
const addError = ref('')
const adding = ref(false)

const cultures = [
  { value: 'default', label: 'Default' },
  { value: 'philippines', label: 'Philippines' },
  { value: 'chinese', label: 'Chinese' },
  { value: 'japanese', label: 'Japanese' },
  { value: 'western', label: 'Western' },
]

async function loadEmployees() {
  try {
    const res = await listEmployees()
    employees.value = res.data
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function handleAdd() {
  addError.value = ''
  adding.value = true
  try {
    await createEmployee(newName.value, newCulture.value)
    newName.value = ''
    newCulture.value = 'default'
    showAdd.value = false
    loading.value = true
    await loadEmployees()
  } catch (e: any) {
    addError.value = e.message
  } finally {
    adding.value = false
  }
}

onMounted(loadEmployees)
</script>

<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem">
      <h2>Team Members</h2>
      <button class="btn btn-primary" @click="showAdd = !showAdd">
        {{ showAdd ? 'Cancel' : '+ Add Employee' }}
      </button>
    </div>

    <div v-if="showAdd" class="card">
      <h3>Add New Employee</h3>
      <form @submit.prevent="handleAdd" style="display: flex; gap: 0.75rem; align-items: flex-end; flex-wrap: wrap">
        <div>
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Name</label>
          <input v-model="newName" placeholder="Employee name" required />
        </div>
        <div>
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Culture</label>
          <select v-model="newCulture">
            <option v-for="c in cultures" :key="c.value" :value="c.value">{{ c.label }}</option>
          </select>
        </div>
        <button type="submit" class="btn btn-primary" :disabled="adding">
          {{ adding ? 'Adding...' : 'Add' }}
        </button>
      </form>
      <p v-if="addError" class="error-msg">{{ addError }}</p>
    </div>

    <p v-if="loading" class="loading">Loading...</p>
    <p v-else-if="error" class="error-msg">{{ error }}</p>
    <div v-else class="card">
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Culture</th>
            <th>Role</th>
            <th>Telegram</th>
            <th>Invite Code</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="employees.length === 0">
            <td colspan="5" style="text-align: center; color: #888; padding: 2rem">
              No employees yet. Add your first team member above.
            </td>
          </tr>
          <tr v-for="emp in employees" :key="emp.id">
            <td><strong>{{ emp.name }}</strong></td>
            <td>{{ emp.culture_code }}</td>
            <td>{{ emp.role }}</td>
            <td>
              <span :class="emp.has_telegram ? 'badge badge-positive' : 'badge badge-neutral'">
                {{ emp.has_telegram ? 'Connected' : 'Pending' }}
              </span>
            </td>
            <td><code>{{ emp.invite_code || '-' }}</code></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
