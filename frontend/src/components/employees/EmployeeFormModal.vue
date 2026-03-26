<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { NModal, NCard, NForm, NFormItem, NInput, NSelect, NButton, NSpace, type FormInst, type FormRules } from 'naive-ui'
import type { Employee } from '@/types'

const props = defineProps<{
  show: boolean
  employee: Employee | null
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  save: [data: { name: string; culture_code: string; job_title: string; responsibilities: string; country: string; language: string }]
}>()

const formRef = ref<FormInst | null>(null)
const saving = ref(false)

const formData = ref({
  name: '',
  culture_code: 'default',
  job_title: '',
  responsibilities: '',
  country: '',
  language: '',
})

const isEdit = computed(() => !!props.employee)
const title = computed(() => (isEdit.value ? 'Edit Employee' : 'Add Employee'))

const cultureOptions = [
  { label: 'Default', value: 'default' },
  { label: 'Philippines', value: 'philippines' },
  { label: 'Singapore', value: 'singapore' },
  { label: 'Indonesia', value: 'indonesia' },
  { label: 'Sri Lanka', value: 'srilanka' },
  { label: 'Malaysia', value: 'malaysia' },
  { label: 'China', value: 'china' },
]

const rules: FormRules = {
  name: { required: true, message: 'Name is required', trigger: 'blur' },
  culture_code: { required: true, message: 'Culture is required', trigger: 'change' },
}

watch(
  () => props.show,
  (visible) => {
    if (visible && props.employee) {
      formData.value = {
        name: props.employee.name,
        culture_code: props.employee.culture_code,
        job_title: props.employee.job_title || '',
        responsibilities: props.employee.responsibilities || '',
        country: props.employee.country || '',
        language: props.employee.language || '',
      }
    } else if (visible) {
      formData.value = { name: '', culture_code: 'default', job_title: '', responsibilities: '', country: '', language: '' }
    }
  },
)

async function handleSave() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    emit('save', { ...formData.value })
  } finally {
    saving.value = false
  }
}

function handleClose() {
  emit('update:show', false)
}
</script>

<template>
  <NModal :show="show" @update:show="handleClose">
    <NCard :title="title" style="width: 500px; max-width: 90vw" :bordered="false" closable @close="handleClose">
      <NForm ref="formRef" :model="formData" :rules="rules" label-placement="left" label-width="110">
        <NFormItem label="Name" path="name">
          <NInput v-model:value="formData.name" placeholder="Employee name" />
        </NFormItem>
        <NFormItem label="Culture" path="culture_code">
          <NSelect v-model:value="formData.culture_code" :options="cultureOptions" />
        </NFormItem>
        <NFormItem label="Job Title" path="job_title">
          <NInput v-model:value="formData.job_title" placeholder="e.g. Senior Engineer" />
        </NFormItem>
        <NFormItem label="Responsibilities" path="responsibilities">
          <NInput v-model:value="formData.responsibilities" type="textarea" placeholder="Key responsibilities" :rows="2" />
        </NFormItem>
        <NFormItem label="Country" path="country">
          <NInput v-model:value="formData.country" placeholder="e.g. Philippines" />
        </NFormItem>
        <NFormItem label="Language" path="language">
          <NInput v-model:value="formData.language" placeholder="e.g. en, tl, zh" />
        </NFormItem>
      </NForm>
      <template #action>
        <NSpace justify="end">
          <NButton @click="handleClose">Cancel</NButton>
          <NButton type="primary" :loading="saving" @click="handleSave">
            {{ isEdit ? 'Update' : 'Create' }}
          </NButton>
        </NSpace>
      </template>
    </NCard>
  </NModal>
</template>
