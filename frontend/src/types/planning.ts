import type { BoardResponse } from './seat'

// Board Records
export interface BoardRecord {
  id: string
  topic: string
  responses: BoardResponse[]
  synthesis: string
  created_at: string
}

export interface BoardRecordsStorage {
  meta: { version: 1; updated_at: string }
  records: BoardRecord[]
}

// OKR / KPI
export type GoalStatus = 'draft' | 'active' | 'completed' | 'cancelled'
export type GoalCycle = string // "2026-Q1"

export interface KeyResult {
  id: string
  title: string
  target: number
  current_value: number
  unit: string // "%", "count", "$"
  due_date: string | null
}

export interface Objective {
  id: string
  title: string
  description: string
  status: GoalStatus
  cycle: GoalCycle
  owner_id: string | null
  key_results: KeyResult[]
  created_at: string
  updated_at: string
}

export interface GoalSnapshot {
  id: string
  goal_id: string
  overall_progress: number
  snapshot_date: string
  created_at: string
}

export interface GoalsStorage {
  meta: { version: 1; updated_at: string }
  objectives: Objective[]
}
