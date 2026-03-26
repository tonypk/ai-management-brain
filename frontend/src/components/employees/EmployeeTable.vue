<script setup lang="ts">
import { h, computed, ref, type VNode } from 'vue'
import { NDataTable, NButton, NIcon, NSpace, NTag, NInput, type DataTableColumn } from 'naive-ui'
import { CreateOutline as EditIcon, TrashOutline as DeleteIcon } from '@vicons/ionicons5'
import type { Employee } from '@/types'

const props = defineProps<{
  data: Employee[]
}>()

const emit = defineEmits<{
  edit: [employee: Employee]
  delete: [employee: Employee]
}>()

const searchQuery = ref('')

const filteredData = computed(() => {
  if (!searchQuery.value) return props.data
  const q = searchQuery.value.toLowerCase()
  return props.data.filter(
    (e) => e.name.toLowerCase().includes(q) || e.role.toLowerCase().includes(q) || e.culture_code.toLowerCase().includes(q),
  )
})

const columns: DataTableColumn<Employee>[] = [
  { title: 'Name', key: 'name', sorter: 'default' },
  { title: 'Role', key: 'role', sorter: 'default' },
  {
    title: 'Culture',
    key: 'culture_code',
    sorter: 'default',
    render(row) {
      return h(NTag, { size: 'small', bordered: false }, { default: () => row.culture_code })
    },
  },
  {
    title: 'Status',
    key: 'is_active',
    render(row) {
      return h(
        NTag,
        { type: row.is_active ? 'success' : 'default', size: 'small' },
        { default: () => (row.is_active ? 'Active' : 'Inactive') },
      )
    },
  },
  {
    title: 'Channels',
    key: 'has_telegram',
    render(row) {
      const tags: VNode[] = []
      if (row.has_telegram) tags.push(h(NTag, { size: 'tiny', type: 'info' }, { default: () => 'TG' }))
      return tags.length > 0 ? h(NSpace, { size: 4 }, { default: () => tags }) : h('span', { style: 'color: #999' }, '--')
    },
  },
  {
    title: 'Actions',
    key: 'actions',
    width: 120,
    render(row) {
      return h(NSpace, { size: 8 }, {
        default: () => [
          h(
            NButton,
            { size: 'small', quaternary: true, onClick: () => emit('edit', row) },
            { icon: () => h(NIcon, { component: EditIcon }) },
          ),
          h(
            NButton,
            { size: 'small', quaternary: true, type: 'error', onClick: () => emit('delete', row) },
            { icon: () => h(NIcon, { component: DeleteIcon }) },
          ),
        ],
      })
    },
  },
]
</script>

<template>
  <div>
    <NInput
      v-model:value="searchQuery"
      placeholder="Search employees..."
      clearable
      style="margin-bottom: 16px; max-width: 320px"
    />
    <NDataTable
      :columns="columns"
      :data="filteredData"
      :pagination="{ pageSize: 10 }"
      :bordered="false"
      size="small"
    />
  </div>
</template>
