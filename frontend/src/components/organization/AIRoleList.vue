<script setup lang="ts">
import { NCard, NTag, NSpace } from 'naive-ui'
import type { AIRole } from '@/types'

defineProps<{ roles: AIRole[] }>()
</script>

<template>
  <div v-if="roles.length" style="display: grid; gap: 12px">
    <NCard
      v-for="role in roles"
      :key="role.id"
      :bordered="false"
      size="small"
      style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)"
    >
      <div style="display: flex; justify-content: space-between; align-items: center">
        <div>
          <div style="font-weight: 600; font-size: 15px">{{ role.role_id }}</div>
          <div style="color: #666; font-size: 13px">{{ role.title }}</div>
          <div style="color: #999; font-size: 12px; margin-top: 2px">Mentor: {{ role.mentor_id }}</div>
        </div>
        <NSpace :size="8" align="center">
          <NTag v-if="role.pending_count > 0" type="warning" size="small">
            {{ role.pending_count }} pending
          </NTag>
          <NTag :type="role.is_active ? 'success' : 'default'" size="small">
            {{ role.is_active ? 'Active' : 'Inactive' }}
          </NTag>
        </NSpace>
      </div>
    </NCard>
  </div>
  <div v-else style="color: #999; text-align: center; padding: 48px 0">
    No AI roles configured. Activate your plan to create AI roles.
  </div>
</template>
