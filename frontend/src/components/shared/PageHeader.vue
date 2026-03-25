<script setup lang="ts">
import { NBreadcrumb, NBreadcrumbItem } from 'naive-ui'

defineProps<{
  title: string
  breadcrumbs?: { label: string; to?: string }[]
}>()
</script>

<template>
  <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 24px">
    <div>
      <NBreadcrumb v-if="breadcrumbs?.length">
        <NBreadcrumbItem v-for="bc in breadcrumbs" :key="bc.label">
          <router-link v-if="bc.to" :to="bc.to">{{ bc.label }}</router-link>
          <span v-else>{{ bc.label }}</span>
        </NBreadcrumbItem>
      </NBreadcrumb>
      <h1 style="font-size: 24px; font-weight: 700; margin: 4px 0 0">{{ title }}</h1>
    </div>
    <div>
      <slot name="actions" />
    </div>
  </div>
</template>
