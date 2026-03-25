<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'

const props = defineProps<{
  sentiment?: string
}>()

const typeMap: Record<string, 'success' | 'error' | 'warning' | 'info' | 'default'> = {
  positive: 'success',
  negative: 'error',
  mixed: 'warning',
  neutral: 'info',
}

const tagType = computed(() => typeMap[props.sentiment || ''] || 'default')
const label = computed(() => props.sentiment || 'unknown')
</script>

<template>
  <NTag :type="tagType" size="small" round>
    {{ label }}
  </NTag>
</template>
