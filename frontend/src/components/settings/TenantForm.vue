<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NCard, NForm, NFormItem, NInput, NSelect, NButton, useMessage } from 'naive-ui'
import { getTenant, updateTenant } from '@/api/settings'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const name = ref('')
const timezone = ref('Asia/Manila')

const tzOptions = [
  'UTC', 'Asia/Manila', 'Asia/Singapore', 'Asia/Tokyo', 'Asia/Shanghai',
  'Asia/Colombo', 'Asia/Jakarta', 'Asia/Kuala_Lumpur', 'America/New_York',
  'America/Los_Angeles', 'Europe/London',
].map(tz => ({ label: tz, value: tz }))

onMounted(async () => {
  loading.value = true
  try {
    const tenant = await getTenant()
    name.value = tenant.name
    timezone.value = tenant.timezone
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to load tenant')
  } finally {
    loading.value = false
  }
})

async function handleSave(): Promise<void> {
  saving.value = true
  try {
    await updateTenant(name.value, timezone.value)
    message.success('Tenant updated')
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : 'Failed to update')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <NCard title="Organization" :bordered="false">
    <NForm label-placement="left" label-width="100">
      <NFormItem label="Name">
        <NInput v-model:value="name" placeholder="Organization name" :disabled="loading" />
      </NFormItem>
      <NFormItem label="Timezone">
        <NSelect v-model:value="timezone" :options="tzOptions" :disabled="loading" />
      </NFormItem>
      <NFormItem>
        <NButton type="primary" :loading="saving" @click="handleSave">Save</NButton>
      </NFormItem>
    </NForm>
  </NCard>
</template>
