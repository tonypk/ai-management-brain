<script setup lang="ts">
import { NCard } from 'naive-ui'
import EmptyState from '@/components/shared/EmptyState.vue'
</script>

<template>
  <NCard :bordered="false">
    <EmptyState description="Alert history is not yet available. Active alerts are shown in the Active tab." />
  </NCard>
</template>
