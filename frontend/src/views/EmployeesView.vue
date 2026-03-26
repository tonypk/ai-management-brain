<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NCard, NButton, NSpin, NIcon, useDialog, useMessage } from 'naive-ui'
import { AddOutline as AddIcon } from '@vicons/ionicons5'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmployeeTable from '@/components/employees/EmployeeTable.vue'
import EmployeeFormModal from '@/components/employees/EmployeeFormModal.vue'
import { listEmployees, createEmployee, updateEmployee, deleteEmployee } from '@/api'
import type { Employee } from '@/types'

const message = useMessage()
const dialog = useDialog()

const loading = ref(true)
const employees = ref<Employee[]>([])
const showModal = ref(false)
const editingEmployee = ref<Employee | null>(null)

async function fetchEmployees() {
  loading.value = true
  try {
    employees.value = await listEmployees()
  } catch (err: unknown) {
    message.error(`Failed to load employees: ${err instanceof Error ? err.message : 'Unknown error'}`)
  } finally {
    loading.value = false
  }
}

function handleAdd() {
  editingEmployee.value = null
  showModal.value = true
}

function handleEdit(employee: Employee) {
  editingEmployee.value = employee
  showModal.value = true
}

function handleDelete(employee: Employee) {
  dialog.warning({
    title: 'Delete Employee',
    content: `Are you sure you want to delete "${employee.name}"? This action cannot be undone.`,
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await deleteEmployee(employee.id)
        message.success('Employee deleted')
        await fetchEmployees()
      } catch (err: unknown) {
        message.error(`Failed to delete: ${err instanceof Error ? err.message : 'Unknown error'}`)
      }
    },
  })
}

async function handleSave(data: { name: string; culture_code: string; job_title: string; responsibilities: string; country: string; language: string }) {
  try {
    if (editingEmployee.value) {
      await updateEmployee(editingEmployee.value.id, data)
      message.success('Employee updated')
    } else {
      await createEmployee(data)
      message.success('Employee created')
    }
    showModal.value = false
    await fetchEmployees()
  } catch (err: unknown) {
    message.error(`Failed to save: ${err instanceof Error ? err.message : 'Unknown error'}`)
  }
}

onMounted(fetchEmployees)
</script>

<template>
  <div>
    <PageHeader title="Team Members">
      <template #actions>
        <NButton type="primary" @click="handleAdd">
          <template #icon>
            <NIcon :component="AddIcon" />
          </template>
          Add Employee
        </NButton>
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <NCard :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
        <EmployeeTable :data="employees" @edit="handleEdit" @delete="handleDelete" />
      </NCard>
    </NSpin>

    <EmployeeFormModal
      v-model:show="showModal"
      :employee="editingEmployee"
      @save="handleSave"
    />
  </div>
</template>
