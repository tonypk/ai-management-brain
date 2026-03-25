import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { useLocalStorage } from '@/composables'
import type {
  BoardRecord, BoardRecordsStorage,
  Objective, KeyResult, GoalStatus, GoalCycle,
} from '@/types'
import * as goalsApi from '@/api/goals'

const BOARD_KEY = 'brain_board_records'

function now(): string {
  return new Date().toISOString()
}

function uid(): string {
  return crypto.randomUUID()
}

function defaultBoardStorage(): BoardRecordsStorage {
  return { meta: { version: 1, updated_at: now() }, records: [] }
}

export const usePlanningStore = defineStore('planning', () => {
  // ── State ──
  const boardData = useLocalStorage<BoardRecordsStorage>(BOARD_KEY, defaultBoardStorage())

  // Goals — backed by API
  const objectives = ref<Objective[]>([])
  const goalsLoading = ref(false)
  const currentCycle = ref('')

  // ── Board Records (localStorage — unchanged) ──
  const boardRecords = computed(() => boardData.value.records)

  function addBoardRecord(topic: string, responses: BoardRecord['responses'], synthesis: string): BoardRecord {
    const record: BoardRecord = {
      id: uid(),
      topic,
      responses,
      synthesis,
      created_at: now(),
    }
    boardData.value = {
      meta: { version: 1, updated_at: now() },
      records: [record, ...boardData.value.records],
    }
    return record
  }

  function deleteBoardRecord(id: string): void {
    boardData.value = {
      meta: { version: 1, updated_at: now() },
      records: boardData.value.records.filter((r) => r.id !== id),
    }
  }

  function searchBoardRecords(query: string): BoardRecord[] {
    if (!query.trim()) return boardData.value.records
    const q = query.toLowerCase()
    return boardData.value.records.filter(
      (r) => r.topic.toLowerCase().includes(q) || r.synthesis.toLowerCase().includes(q),
    )
  }

  // ── Goals / OKR (API-backed) ──

  async function loadGoals(cycle: string): Promise<void> {
    goalsLoading.value = true
    try {
      currentCycle.value = cycle
      objectives.value = await goalsApi.listGoals(cycle)
    } finally {
      goalsLoading.value = false
    }
  }

  function objectivesByCycle(cycle: GoalCycle): Objective[] {
    return objectives.value.filter((o) => o.cycle === cycle)
  }

  async function addObjective(
    title: string,
    description: string,
    cycle: GoalCycle,
    status: GoalStatus = 'draft',
    ownerId?: string | null,
  ): Promise<Objective> {
    const goal = await goalsApi.createGoal({ title, description, cycle, owner_id: ownerId, status })
    await loadGoals(currentCycle.value)
    return goal
  }

  async function updateObjective(
    id: string,
    patch: Partial<Pick<Objective, 'title' | 'description' | 'status' | 'cycle' | 'owner_id'>>,
  ): Promise<void> {
    const existing = objectives.value.find((o) => o.id === id)
    if (!existing) return
    await goalsApi.updateGoal(id, {
      title: patch.title ?? existing.title,
      description: patch.description ?? existing.description,
      cycle: patch.cycle ?? existing.cycle,
      status: patch.status ?? existing.status,
      owner_id: patch.owner_id !== undefined ? patch.owner_id : existing.owner_id,
    })
    await loadGoals(currentCycle.value)
  }

  async function deleteObjective(id: string): Promise<void> {
    await goalsApi.deleteGoal(id)
    await loadGoals(currentCycle.value)
  }

  async function addKeyResult(
    objectiveId: string,
    title: string,
    target: number,
    unit: string,
    dueDate: string,
  ): Promise<void> {
    await goalsApi.createKeyResult(objectiveId, {
      title,
      target,
      unit,
      due_date: dueDate || null,
    })
    await loadGoals(currentCycle.value)
  }

  async function updateKeyResult(
    objectiveId: string,
    krId: string,
    patch: Partial<Pick<KeyResult, 'title' | 'target' | 'current_value' | 'unit' | 'due_date'>>,
  ): Promise<void> {
    const obj = objectives.value.find((o) => o.id === objectiveId)
    const kr = obj?.key_results.find((k) => k.id === krId)
    if (!kr) return
    await goalsApi.updateKeyResult(objectiveId, krId, {
      title: patch.title ?? kr.title,
      target: patch.target ?? kr.target,
      current_value: patch.current_value ?? kr.current_value,
      unit: patch.unit ?? kr.unit,
      due_date: patch.due_date !== undefined ? patch.due_date : kr.due_date,
    })
    await loadGoals(currentCycle.value)
  }

  async function deleteKeyResult(objectiveId: string, krId: string): Promise<void> {
    await goalsApi.deleteKeyResult(objectiveId, krId)
    await loadGoals(currentCycle.value)
  }

  // ── Computed Stats ──
  function cycleStats(cycle: GoalCycle) {
    const objs = objectivesByCycle(cycle)
    const total = objs.length
    const active = objs.filter((o) => o.status === 'active').length
    const completed = objs.filter((o) => o.status === 'completed').length

    let progress = 0
    if (total > 0) {
      const objProgresses = objs.map((o) => {
        if (o.key_results.length === 0) return 0
        const krSum = o.key_results.reduce((acc, kr) => {
          const p = kr.target > 0 ? Math.min((kr.current_value / kr.target) * 100, 100) : 0
          return acc + p
        }, 0)
        return krSum / o.key_results.length
      })
      progress = Math.round(objProgresses.reduce((a, b) => a + b, 0) / total)
    }

    return { total, active, completed, progress }
  }

  return {
    // board
    boardRecords,
    addBoardRecord,
    deleteBoardRecord,
    searchBoardRecords,
    // goals
    objectives,
    goalsLoading,
    currentCycle,
    loadGoals,
    objectivesByCycle,
    addObjective,
    updateObjective,
    deleteObjective,
    addKeyResult,
    updateKeyResult,
    deleteKeyResult,
    cycleStats,
  }
})
