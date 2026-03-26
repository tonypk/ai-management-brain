<script setup lang="ts">
import { computed } from 'vue'
import { NCard, NTag, NSpace } from 'naive-ui'
import type { Employee } from '@/types'

const props = defineProps<{
  employee: Employee
}>()

const initials = computed(() => {
  const parts = props.employee.name.split(' ')
  return parts.map(p => p[0]).join('').toUpperCase().slice(0, 2)
})

const cultureLabel: Record<string, string> = {
  default: 'Default',
  philippines: 'Philippines',
  singapore: 'Singapore',
  indonesia: 'Indonesia',
  srilanka: 'Sri Lanka',
  malaysia: 'Malaysia',
  china: 'China',
}
</script>

<template>
  <NCard :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <div style="display: flex; align-items: center; gap: 20px">
      <div style="width: 64px; height: 64px; border-radius: 50%; background: #6366f1; color: #fff; display: flex; align-items: center; justify-content: center; font-size: 22px; font-weight: 700; flex-shrink: 0">
        {{ initials }}
      </div>
      <div style="flex: 1; min-width: 0">
        <div style="display: flex; align-items: center; gap: 8px; flex-wrap: wrap">
          <span style="font-size: 20px; font-weight: 700">{{ employee.name }}</span>
          <NTag v-if="employee.job_title" type="info" size="small" round>
            {{ employee.job_title }}
          </NTag>
          <NTag v-if="employee.culture_code" size="small" round>
            {{ cultureLabel[employee.culture_code] || employee.culture_code }}
          </NTag>
        </div>
        <div v-if="employee.responsibilities" style="color: #666; font-size: 13px; margin-top: 4px">
          {{ employee.responsibilities }}
        </div>
        <NSpace v-if="employee.country || employee.language" :size="12" style="margin-top: 6px">
          <span v-if="employee.country" style="font-size: 12px; color: #888">{{ employee.country }}</span>
          <span v-if="employee.language" style="font-size: 12px; color: #888">{{ employee.language }}</span>
        </NSpace>
      </div>
    </div>
  </NCard>
</template>
