<script setup lang="ts">
import { NDataTable, NTag, NProgress } from 'naive-ui'
import { computed, h } from 'vue'
import type { SkillRow } from '@/api/worldmodel'

const props = defineProps<{ skills: SkillRow[] }>()

const columns = [
  { title: 'Employee', key: 'employee_name', sorter: 'default' },
  { title: 'Skill', key: 'skill_name', sorter: 'default' },
  {
    title: 'Proficiency',
    key: 'proficiency',
    render: (row: SkillRow) => {
      const colors: Record<string, string> = { expert: 'success', high: 'info', medium: 'warning', low: 'error' }
      return h(NTag, { type: colors[row.proficiency] || 'default', size: 'small' }, () => row.proficiency)
    },
  },
  {
    title: 'Confidence',
    key: 'confidence',
    sorter: (a: SkillRow, b: SkillRow) => a.confidence - b.confidence,
    render: (row: SkillRow) => h(NProgress, { percentage: Math.round(row.confidence * 100), type: 'line', showIndicator: true }),
  },
  { title: 'Mentions', key: 'mention_count', sorter: 'default' },
]

const data = computed(() => props.skills)
</script>

<template>
  <NDataTable :columns="columns" :data="data" :pagination="{ pageSize: 20 }" size="small" />
</template>
