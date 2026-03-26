<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  NButton, NIcon, NCard, NSpin, NDataTable, NTag,
  NModal, NInput, NSelect, NSpace, NFormItem, NRate,
  useMessage, useDialog,
} from 'naive-ui'
import { AddOutline, TrashOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import {
  listSkills, createSkill, deleteSkill,
  getSkillMatrix, setEmployeeSkill,
} from '@/api/skills'
import { listEmployees } from '@/api/employees'
import type { Skill, SkillMatrixEntry, Employee } from '@/types'

const message = useMessage()
const dialog = useDialog()

const loading = ref(true)
const skills = ref<Skill[]>([])
const employees = ref<Employee[]>([])
const matrix = ref<SkillMatrixEntry[]>([])

// Skill modal
const showSkillModal = ref(false)
const skillForm = ref({ name: '', category: '', description: '' })

// Assign skill modal
const showAssignModal = ref(false)
const assignForm = ref({ employee_id: '', skill_id: '', level: 3, notes: '' })

onMounted(async () => {
  try {
    const [s, e, m] = await Promise.all([listSkills(), listEmployees(), getSkillMatrix()])
    skills.value = s
    employees.value = e
    matrix.value = m
  } catch {
    message.error('Failed to load data')
  } finally {
    loading.value = false
  }
})

async function handleCreateSkill() {
  try {
    const s = await createSkill(skillForm.value)
    skills.value.push({ ...s, employee_count: 0 })
    showSkillModal.value = false
    skillForm.value = { name: '', category: '', description: '' }
    message.success('Skill created')
  } catch {
    message.error('Failed to create skill')
  }
}

function handleDeleteSkill(id: string) {
  dialog.warning({
    title: 'Delete Skill',
    content: 'This will remove the skill from all employees.',
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      await deleteSkill(id)
      skills.value = skills.value.filter((s) => s.id !== id)
      matrix.value = matrix.value.filter((m) => m.skill_id !== id)
      message.success('Skill deleted')
    },
  })
}

async function handleAssignSkill() {
  try {
    await setEmployeeSkill(assignForm.value.employee_id, {
      skill_id: assignForm.value.skill_id,
      level: assignForm.value.level,
      notes: assignForm.value.notes,
    })
    // Refresh matrix
    matrix.value = await getSkillMatrix()
    showAssignModal.value = false
    message.success('Skill assigned')
  } catch {
    message.error('Failed to assign skill')
  }
}

const employeeOptions = computed(() =>
  employees.value.map((e) => ({ label: e.name, value: e.id }))
)

const skillOptions = computed(() =>
  skills.value.map((s) => ({ label: `${s.name} (${s.category})`, value: s.id }))
)

// Build matrix table: rows = employees, columns = skills
const matrixEmployees = computed(() => [...new Set(matrix.value.map((m) => m.employee_name))].sort())
const matrixSkills = computed(() => {
  const seen = new Set<string>()
  return matrix.value.filter((m) => {
    if (seen.has(m.skill_id)) return false
    seen.add(m.skill_id)
    return true
  }).map((m) => ({ id: m.skill_id, name: m.skill_name, category: m.category }))
})

function matrixLevel(empName: string, skillId: string): number {
  return matrix.value.find((m) => m.employee_name === empName && m.skill_id === skillId)?.level ?? 0
}

function levelColor(level: number): string {
  if (level >= 4) return '#22c55e'
  if (level >= 3) return '#f59e0b'
  if (level >= 1) return '#ef4444'
  return '#e5e5e5'
}

const skillColumns: DataTableColumns<Skill> = [
  { title: 'Skill', key: 'name' },
  { title: 'Category', key: 'category', render: (s) => h(NTag, { size: 'small' }, () => s.category) },
  { title: 'Employees', key: 'employee_count', width: 90 },
  {
    title: '', key: 'action', width: 60,
    render: (s) => h(NButton, { size: 'tiny', type: 'error', quaternary: true, onClick: () => handleDeleteSkill(s.id) },
      () => h(NIcon, { component: TrashOutline })),
  },
]
</script>

<script lang="ts">
import { h } from 'vue'
export default {}
</script>

<template>
  <div>
    <PageHeader title="Skill Inventory">
      <template #actions>
        <NSpace :size="8">
          <NButton @click="showAssignModal = true; assignForm = { employee_id: '', skill_id: '', level: 3, notes: '' }">
            Assign Skill
          </NButton>
          <NButton type="primary" @click="showSkillModal = true; skillForm = { name: '', category: '', description: '' }">
            <template #icon><NIcon :component="AddOutline" /></template>
            New Skill
          </NButton>
        </NSpace>
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <!-- Skills Table -->
      <NCard :bordered="false" size="small" style="margin-bottom: 20px">
        <div style="font-weight: 600; font-size: 14px; margin-bottom: 8px">Skills ({{ skills.length }})</div>
        <EmptyState v-if="skills.length === 0" description="No skills defined yet" />
        <NDataTable v-else :columns="skillColumns" :data="skills" :bordered="false" size="small" />
      </NCard>

      <!-- Skill Matrix -->
      <NCard v-if="matrixEmployees.length > 0 && matrixSkills.length > 0" :bordered="false" size="small">
        <div style="font-weight: 600; font-size: 14px; margin-bottom: 8px">Skill Matrix</div>
        <div style="overflow-x: auto">
          <table style="width: 100%; border-collapse: collapse; font-size: 13px">
            <thead>
              <tr>
                <th style="text-align: left; padding: 8px; border-bottom: 2px solid #eee">Employee</th>
                <th v-for="s in matrixSkills" :key="s.id" style="text-align: center; padding: 8px; border-bottom: 2px solid #eee; white-space: nowrap">
                  {{ s.name }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="emp in matrixEmployees" :key="emp">
                <td style="padding: 6px 8px; border-bottom: 1px solid #f5f5f5">{{ emp }}</td>
                <td v-for="s in matrixSkills" :key="s.id" style="text-align: center; padding: 6px; border-bottom: 1px solid #f5f5f5">
                  <div
                    v-if="matrixLevel(emp, s.id) > 0"
                    :style="{ display: 'inline-block', width: '24px', height: '24px', borderRadius: '4px', backgroundColor: levelColor(matrixLevel(emp, s.id)), color: '#fff', lineHeight: '24px', fontSize: '12px', fontWeight: 600 }"
                  >
                    {{ matrixLevel(emp, s.id) }}
                  </div>
                  <span v-else style="color: #ccc">—</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </NCard>
    </NSpin>

    <!-- Create Skill Modal -->
    <NModal v-model:show="showSkillModal" preset="card" title="New Skill" style="max-width: 420px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Name" :show-feedback="false">
          <NInput v-model:value="skillForm.name" placeholder="e.g. Python" />
        </NFormItem>
        <NFormItem label="Category" :show-feedback="false">
          <NInput v-model:value="skillForm.category" placeholder="e.g. Programming" />
        </NFormItem>
        <NFormItem label="Description" :show-feedback="false">
          <NInput v-model:value="skillForm.description" type="textarea" :rows="2" />
        </NFormItem>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showSkillModal = false">Cancel</NButton>
          <NButton type="primary" :disabled="!skillForm.name.trim()" @click="handleCreateSkill">Create</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Assign Skill Modal -->
    <NModal v-model:show="showAssignModal" preset="card" title="Assign Skill" style="max-width: 420px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Employee" :show-feedback="false">
          <NSelect v-model:value="assignForm.employee_id" :options="employeeOptions" placeholder="Select employee" />
        </NFormItem>
        <NFormItem label="Skill" :show-feedback="false">
          <NSelect v-model:value="assignForm.skill_id" :options="skillOptions" placeholder="Select skill" />
        </NFormItem>
        <NFormItem label="Level (1-5)" :show-feedback="false">
          <NRate v-model:value="assignForm.level" :count="5" />
        </NFormItem>
        <NFormItem label="Notes" :show-feedback="false">
          <NInput v-model:value="assignForm.notes" placeholder="Optional notes" />
        </NFormItem>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showAssignModal = false">Cancel</NButton>
          <NButton type="primary" :disabled="!assignForm.employee_id || !assignForm.skill_id" @click="handleAssignSkill">Assign</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
