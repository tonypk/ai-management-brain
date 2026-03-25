<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { NButton, NIcon, NGrid, NGi, NCard, NSpin, useMessage } from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import GoalOverviewStats from '@/components/goals/GoalOverviewStats.vue'
import GoalCycleSelector from '@/components/goals/GoalCycleSelector.vue'
import GoalProgressChart from '@/components/goals/GoalProgressChart.vue'
import GoalDeviationChart from '@/components/goals/GoalDeviationChart.vue'
import ObjectiveCard from '@/components/goals/ObjectiveCard.vue'
import ObjectiveFormModal from '@/components/goals/ObjectiveFormModal.vue'
import { usePlanningStore } from '@/stores/planning'
import type { Objective, GoalStatus } from '@/types'

const store = usePlanningStore()
const message = useMessage()

function currentCycleKey(): string {
  const now = new Date()
  return `${now.getFullYear()}-Q${Math.ceil((now.getMonth() + 1) / 3)}`
}

const selectedCycle = ref(currentCycleKey())
const showObjectiveModal = ref(false)
const editingObjective = ref<Objective | null>(null)

const filteredObjectives = computed(() => store.objectivesByCycle(selectedCycle.value))
const stats = computed(() => store.cycleStats(selectedCycle.value))

// Load goals from API
onMounted(() => store.loadGoals(selectedCycle.value))
watch(selectedCycle, (cycle) => store.loadGoals(cycle))

function handleNewObjective() {
  editingObjective.value = null
  showObjectiveModal.value = true
}

function handleEditObjective(obj: Objective) {
  editingObjective.value = obj
  showObjectiveModal.value = true
}

async function handleObjectiveSubmit(data: { title: string; description: string; status: GoalStatus; cycle: string }) {
  if (editingObjective.value) {
    await store.updateObjective(editingObjective.value.id, data)
    message.success('Objective updated')
  } else {
    await store.addObjective(data.title, data.description, data.cycle, data.status)
    message.success('Objective created')
  }
}
</script>

<template>
  <div>
    <PageHeader title="Goals & KPIs">
      <template #actions>
        <NButton type="primary" @click="handleNewObjective">
          <template #icon><NIcon :component="AddOutline" /></template>
          New Objective
        </NButton>
      </template>
    </PageHeader>

    <NSpin :show="store.goalsLoading">
      <GoalOverviewStats
        :progress="stats.progress"
        :total="stats.total"
        :active="stats.active"
        :completed="stats.completed"
      />

      <div style="display: flex; align-items: center; gap: 12px; margin: 20px 0 16px">
        <span style="font-weight: 600; font-size: 14px">Cycle:</span>
        <GoalCycleSelector v-model="selectedCycle" />
      </div>

      <NCard v-if="filteredObjectives.length > 0" :bordered="false" size="small" style="margin-bottom: 20px">
        <div style="font-weight: 600; font-size: 14px; margin-bottom: 8px">Progress Overview</div>
        <GoalProgressChart :objectives="filteredObjectives" />
      </NCard>

      <NCard v-if="filteredObjectives.length > 0" :bordered="false" size="small" style="margin-bottom: 20px">
        <div style="font-weight: 600; font-size: 14px; margin-bottom: 8px">Deviation Tracking</div>
        <GoalDeviationChart :objectives="filteredObjectives" />
      </NCard>

      <EmptyState v-if="!store.goalsLoading && filteredObjectives.length === 0" description="No objectives for this cycle" />
      <NGrid v-else :x-gap="16" :y-gap="16" cols="1 m:2" responsive="screen">
        <NGi v-for="obj in filteredObjectives" :key="obj.id">
          <ObjectiveCard :objective="obj" @edit="handleEditObjective" />
        </NGi>
      </NGrid>
    </NSpin>

    <ObjectiveFormModal
      v-model:show="showObjectiveModal"
      :objective="editingObjective"
      :default-cycle="selectedCycle"
      @submit="handleObjectiveSubmit"
    />
  </div>
</template>
