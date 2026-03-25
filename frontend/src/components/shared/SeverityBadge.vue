<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'

const props = defineProps<{
  severity: 'warning' | 'critical' | string
}>()

const tagType = computed(() => {
  if (props.severity === 'critical') return 'error' as const
  if (props.severity === 'warning') return 'warning' as const
  return 'default' as const
})
</script>

<template>
  <NTag :type="tagType" size="small" round>
    {{ severity }}
  </NTag>
</template>
