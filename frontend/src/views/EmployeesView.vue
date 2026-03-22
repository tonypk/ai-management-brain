<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listEmployees, createEmployee, updateEmployeeProfile, type Employee } from '../composables/api'

const employees = ref<Employee[]>([])
const loading = ref(true)
const error = ref('')

const showAdd = ref(false)
const newName = ref('')
const newCulture = ref('default')
const newJobTitle = ref('')
const newResponsibilities = ref('')
const newCountry = ref('')
const newLanguage = ref('')
const addError = ref('')
const adding = ref(false)

// Edit modal state
const showEdit = ref(false)
const editEmployee = ref<Employee | null>(null)
const editJobTitle = ref('')
const editResponsibilities = ref('')
const editCountry = ref('')
const editLanguage = ref('')
const editError = ref('')
const saving = ref(false)

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
    await createEmployee({
      name: newName.value,
      culture_code: newCulture.value,
      job_title: newJobTitle.value || undefined,
      responsibilities: newResponsibilities.value || undefined,
      country: newCountry.value || undefined,
      language: newLanguage.value || undefined,
    })
    newName.value = ''
    newCulture.value = 'default'
    newJobTitle.value = ''
    newResponsibilities.value = ''
    newCountry.value = ''
    newLanguage.value = ''
    showAdd.value = false
    loading.value = true
    await loadEmployees()
  } catch (e: any) {
    addError.value = e.message
  } finally {
    adding.value = false
  }
}

function openEdit(emp: Employee) {
  editEmployee.value = emp
  editJobTitle.value = emp.job_title || ''
  editResponsibilities.value = emp.responsibilities || ''
  editCountry.value = emp.country || ''
  editLanguage.value = emp.language || ''
  editError.value = ''
  showEdit.value = true
}

async function handleSaveEdit() {
  if (!editEmployee.value) return
  editError.value = ''
  saving.value = true
  try {
    await updateEmployeeProfile(editEmployee.value.id, {
      job_title: editJobTitle.value,
      responsibilities: editResponsibilities.value,
      country: editCountry.value,
      language: editLanguage.value,
    })
    showEdit.value = false
    loading.value = true
    await loadEmployees()
  } catch (e: any) {
    editError.value = e.message
  } finally {
    saving.value = false
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
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Name *</label>
          <input v-model="newName" placeholder="Employee name" required />
        </div>
        <div>
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Culture</label>
          <select v-model="newCulture">
            <option v-for="c in cultures" :key="c.value" :value="c.value">{{ c.label }}</option>
          </select>
        </div>
        <div>
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Job Title</label>
          <input v-model="newJobTitle" placeholder="e.g. Frontend Developer" />
        </div>
        <div>
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Country</label>
          <input v-model="newCountry" placeholder="e.g. Philippines" />
        </div>
        <div>
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Language</label>
          <input v-model="newLanguage" placeholder="e.g. Chinese" />
        </div>
        <div style="width: 100%">
          <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Responsibilities</label>
          <textarea v-model="newResponsibilities" placeholder="Brief description of role" rows="2" style="width: 100%"></textarea>
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
            <th>Job Title</th>
            <th>Culture</th>
            <th>Role</th>
            <th>Telegram</th>
            <th>Invite Code</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="employees.length === 0">
            <td colspan="7" style="text-align: center; color: #888; padding: 2rem">
              No employees yet. Add your first team member above.
            </td>
          </tr>
          <tr v-for="emp in employees" :key="emp.id">
            <td><strong>{{ emp.name }}</strong></td>
            <td>{{ emp.job_title || '-' }}</td>
            <td>{{ emp.culture_code }}</td>
            <td>{{ emp.role }}</td>
            <td>
              <span :class="emp.has_telegram ? 'badge badge-positive' : 'badge badge-neutral'">
                {{ emp.has_telegram ? 'Connected' : 'Pending' }}
              </span>
            </td>
            <td><code>{{ emp.invite_code || '-' }}</code></td>
            <td>
              <button class="btn" style="font-size: 0.8rem; padding: 0.25rem 0.5rem" @click="openEdit(emp)">Edit</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Edit Profile Modal -->
    <div v-if="showEdit" style="position: fixed; inset: 0; background: rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; z-index: 100" @click.self="showEdit = false">
      <div class="card" style="width: 100%; max-width: 500px; margin: 1rem">
        <h3>Edit Profile: {{ editEmployee?.name }}</h3>
        <form @submit.prevent="handleSaveEdit" style="display: flex; flex-direction: column; gap: 0.75rem">
          <div>
            <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Job Title</label>
            <input v-model="editJobTitle" placeholder="e.g. Frontend Developer" style="width: 100%" />
          </div>
          <div>
            <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Responsibilities</label>
            <textarea v-model="editResponsibilities" placeholder="Brief description" rows="3" style="width: 100%"></textarea>
          </div>
          <div>
            <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Country</label>
            <input v-model="editCountry" placeholder="e.g. Philippines" style="width: 100%" />
          </div>
          <div>
            <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Language</label>
            <input v-model="editLanguage" placeholder="e.g. Chinese" style="width: 100%" />
          </div>
          <div style="display: flex; gap: 0.5rem; justify-content: flex-end">
            <button type="button" class="btn" @click="showEdit = false">Cancel</button>
            <button type="submit" class="btn btn-primary" :disabled="saving">
              {{ saving ? 'Saving...' : 'Save' }}
            </button>
          </div>
          <p v-if="editError" class="error-msg">{{ editError }}</p>
        </form>
      </div>
    </div>
  </div>
</template>
