<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  NButton, NIcon, NSpin, NDataTable, NTag,
  NModal, NInput, NSelect, NSpace, NFormItem,
  useMessage,
} from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import {
  createMetric, deleteMetric,
  getMetricsWithValues, ingestMetricValue,
} from '@/api/metrics'
import { listEmployees } from '@/api/employees'
import type { MetricWithValue, Employee } from '@/types'

const message = useMessage()

const loading = ref(true)
const metrics = ref<MetricWithValue[]>([])
const employees = ref<Employee[]>([])

const showCreateModal = ref(false)
const form = ref({
  name: '', description: '', unit: '', frequency: 'monthly',
  target_value: '', alert_threshold: '', owner_id: '',
})

const showIngestModal = ref(false)
const ingestForm = ref({ metric_id: '', value: '', source: '' })

onMounted(async () => {
  try {
    const [m, e] = await Promise.all([getMetricsWithValues(), listEmployees()])
    metrics.value = m
    employees.value = e
  } catch {
    message.error('Failed to load metrics')
  } finally {
    loading.value = false
  }
})

async function handleCreate() {
  try {
    await createMetric(form.value)
    metrics.value = await getMetricsWithValues()
    showCreateModal.value = false
    form.value = { name: '', description: '', unit: '', frequency: 'monthly', target_value: '', alert_threshold: '', owner_id: '' }
    message.success('Metric created')
  } catch {
    message.error('Failed to create metric')
  }
}

async function handleDelete(id: string) {
  try {
    await deleteMetric(id)
    metrics.value = metrics.value.filter((m) => m.id !== id)
    message.success('Metric deleted')
  } catch {
    message.error('Failed to delete metric')
  }
}

async function handleIngest() {
  try {
    await ingestMetricValue(ingestForm.value)
    metrics.value = await getMetricsWithValues()
    showIngestModal.value = false
    ingestForm.value = { metric_id: '', value: '', source: '' }
    message.success('Value recorded')
  } catch {
    message.error('Failed to record value')
  }
}

const employeeOptions = computed(() =>
  employees.value.map((e) => ({ label: e.name, value: e.id })),
)

const metricOptions = computed(() =>
  metrics.value.map((m) => ({ label: m.name, value: m.id })),
)

const frequencyOptions = [
  { label: 'Daily', value: 'daily' },
  { label: 'Weekly', value: 'weekly' },
  { label: 'Monthly', value: 'monthly' },
  { label: 'Quarterly', value: 'quarterly' },
]

const columns: DataTableColumns<MetricWithValue> = [
  { title: 'Name', key: 'name' },
  { title: 'Unit', key: 'unit', width: 80, render: (m) => m.unit || '—' },
  { title: 'Frequency', key: 'refresh_frequency', width: 100, render: (m) => m.refresh_frequency || '—' },
  { title: 'Target', key: 'target_value', width: 90, render: (m) => m.target_value || '—' },
  {
    title: 'Latest', key: 'latest_value', width: 90,
    render: (m) => m.latest_value
      ? h(NTag, { size: 'small', type: 'info' }, () => m.latest_value!)
      : '—',
  },
  { title: 'Owner', key: 'owner_name', width: 120, render: (m) => m.owner_name || '—' },
  {
    title: '', key: 'action', width: 60,
    render: (m) => h(NButton, { size: 'tiny', type: 'error', quaternary: true, onClick: () => handleDelete(m.id) }, () => 'Del'),
  },
]
</script>

<script lang="ts">
import { h } from 'vue'
export default {}
</script>

<template>
  <div>
    <PageHeader title="KPI Metrics">
      <template #actions>
        <NSpace :size="8">
          <NButton @click="showIngestModal = true; ingestForm = { metric_id: '', value: '', source: '' }">
            Record Value
          </NButton>
          <NButton type="primary" @click="showCreateModal = true">
            <template #icon><NIcon :component="AddOutline" /></template>
            New Metric
          </NButton>
        </NSpace>
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <EmptyState v-if="metrics.length === 0 && !loading" description="No KPI metrics defined yet" />
      <NDataTable v-else :columns="columns" :data="metrics" :bordered="false" size="small" />
    </NSpin>

    <!-- Create Metric Modal -->
    <NModal v-model:show="showCreateModal" preset="card" title="New KPI Metric" style="max-width: 480px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Name" :show-feedback="false">
          <NInput v-model:value="form.name" placeholder="e.g. Monthly Revenue" />
        </NFormItem>
        <NFormItem label="Description" :show-feedback="false">
          <NInput v-model:value="form.description" type="textarea" :rows="2" />
        </NFormItem>
        <NSpace :size="12">
          <NFormItem label="Unit" :show-feedback="false">
            <NInput v-model:value="form.unit" placeholder="e.g. USD, %" style="width: 100px" />
          </NFormItem>
          <NFormItem label="Frequency" :show-feedback="false">
            <NSelect v-model:value="form.frequency" :options="frequencyOptions" style="width: 130px" />
          </NFormItem>
        </NSpace>
        <NSpace :size="12">
          <NFormItem label="Target Value" :show-feedback="false">
            <NInput v-model:value="form.target_value" placeholder="e.g. 100000" style="width: 130px" />
          </NFormItem>
          <NFormItem label="Alert Threshold" :show-feedback="false">
            <NInput v-model:value="form.alert_threshold" placeholder="e.g. 80000" style="width: 130px" />
          </NFormItem>
        </NSpace>
        <NFormItem label="Owner" :show-feedback="false">
          <NSelect v-model:value="form.owner_id" :options="employeeOptions" placeholder="Select owner" clearable />
        </NFormItem>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCreateModal = false">Cancel</NButton>
          <NButton type="primary" :disabled="!form.name.trim()" @click="handleCreate">Create</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Ingest Value Modal -->
    <NModal v-model:show="showIngestModal" preset="card" title="Record Metric Value" style="max-width: 400px; width: 95%">
      <NSpace vertical :size="12">
        <NFormItem label="Metric" :show-feedback="false">
          <NSelect v-model:value="ingestForm.metric_id" :options="metricOptions" placeholder="Select metric" />
        </NFormItem>
        <NFormItem label="Value" :show-feedback="false">
          <NInput v-model:value="ingestForm.value" placeholder="e.g. 95000" />
        </NFormItem>
        <NFormItem label="Source" :show-feedback="false">
          <NInput v-model:value="ingestForm.source" placeholder="e.g. manual, api, spreadsheet" />
        </NFormItem>
      </NSpace>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showIngestModal = false">Cancel</NButton>
          <NButton type="primary" :disabled="!ingestForm.metric_id || !ingestForm.value" @click="handleIngest">Record</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
