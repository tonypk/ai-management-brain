import { get } from './client'
import type {
  CompanyState, CommunicationEvent, ExecutionSignal,
  WorkingMemorySnapshot,
} from '@/types'

export async function getCompanyState(): Promise<CompanyState> {
  const res = await get<{ data: CompanyState }>('/state')
  return res.data
}

export async function listCommunicationEvents(params?: {
  event_type?: string
  limit?: number
}): Promise<CommunicationEvent[]> {
  const query = new URLSearchParams()
  if (params?.event_type) query.set('event_type', params.event_type)
  if (params?.limit) query.set('limit', String(params.limit))
  const qs = query.toString()
  const res = await get<{ data: CommunicationEvent[] }>(`/state/events${qs ? '?' + qs : ''}`)
  return res.data
}

export async function listExecutionSignals(params?: {
  signal_type?: string
  limit?: number
}): Promise<ExecutionSignal[]> {
  const query = new URLSearchParams()
  if (params?.signal_type) query.set('signal_type', params.signal_type)
  if (params?.limit) query.set('limit', String(params.limit))
  const qs = query.toString()
  const res = await get<{ data: ExecutionSignal[] }>(`/state/signals${qs ? '?' + qs : ''}`)
  return res.data
}

export async function getTopRisks(limit?: number): Promise<ExecutionSignal[]> {
  const qs = limit ? `?limit=${limit}` : ''
  const res = await get<{ data: ExecutionSignal[] }>(`/state/risks${qs}`)
  return res.data
}

export async function getWorkingMemory(): Promise<WorkingMemorySnapshot | null> {
  const res = await get<{ data: WorkingMemorySnapshot | null }>('/state/memory')
  return res.data
}
