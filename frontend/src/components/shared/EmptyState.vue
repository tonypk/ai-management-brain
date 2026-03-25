<script setup lang="ts">
import { NEmpty } from 'naive-ui'

defineProps<{
  description?: string
}>()
</script>

<template>
  <NEmpty :description="description || 'No data'" style="padding: 48px 0" />
</template>
