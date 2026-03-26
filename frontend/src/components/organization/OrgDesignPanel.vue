<script setup lang="ts">
import { NCard, NTag, NSpace, NDivider } from 'naive-ui'
import type { OrgDesign } from '@/types'

defineProps<{ design: OrgDesign }>()
</script>

<template>
  <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px">
    <NCard title="Org Design" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
      <div style="display: flex; gap: 8px; margin-bottom: 12px">
        <NTag type="info">{{ design.philosophy }}</NTag>
        <NTag>{{ design.structure_type }}</NTag>
      </div>
      <NDivider style="margin: 12px 0">Units</NDivider>
      <div v-for="unit in design.units" :key="unit.name" style="padding: 12px; border: 1px solid #e5e7eb; border-radius: 8px; margin-bottom: 8px">
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span style="font-weight: 600">{{ unit.name }}</span>
          <NSpace :size="4">
            <NTag size="small" type="info">{{ unit.leader_role }}</NTag>
            <NTag v-if="unit.size" size="small">{{ unit.size }} people</NTag>
          </NSpace>
        </div>
        <div v-if="unit.kpis?.length" style="margin-top: 6px">
          <NTag v-for="kpi in unit.kpis" :key="kpi" size="small" style="margin-right: 4px; margin-bottom: 4px">
            {{ kpi }}
          </NTag>
        </div>
      </div>
    </NCard>

    <NCard title="Support Roles" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
      <template v-if="design.support_roles?.length">
        <div v-for="role in design.support_roles" :key="role.title" style="padding: 12px; border: 1px solid #e5e7eb; border-radius: 8px; margin-bottom: 8px">
          <div style="display: flex; justify-content: space-between; align-items: center">
            <span style="font-weight: 600">{{ role.title }}</span>
            <NTag :type="role.type === 'ai' ? 'success' : 'default'" size="small">
              {{ role.type === 'ai' ? 'AI' : 'Human' }}
            </NTag>
          </div>
          <div style="color: #666; font-size: 13px; margin-top: 4px">Scope: {{ role.scope }}</div>
        </div>
      </template>
      <div v-else style="color: #999; text-align: center; padding: 24px">No support roles defined</div>
    </NCard>
  </div>
</template>
