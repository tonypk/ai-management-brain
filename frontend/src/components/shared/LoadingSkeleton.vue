<script setup lang="ts">
import { NSkeleton, NSpace } from 'naive-ui'

defineProps<{
  rows?: number
}>()
</script>

<template>
  <NSpace vertical :size="12" style="padding: 16px 0">
    <NSkeleton v-for="i in (rows || 3)" :key="i" text :repeat="1" style="width: 100%" />
  </NSpace>
</template>
