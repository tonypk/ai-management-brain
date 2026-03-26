import { get, post, put, del } from './client'
import type { CareerLevel, CareerPath } from '@/types'

export async function listCareerLevels(): Promise<CareerLevel[]> {
  const res = await get<{ data: CareerLevel[] }>('/career/levels')
  return res.data
}

export async function createCareerLevel(data: {
  title: string
  level_order?: number
  description?: string
  requirements?: string
}): Promise<CareerLevel> {
  const res = await post<{ data: CareerLevel }>('/career/levels', data)
  return res.data
}

export async function updateCareerLevel(
  id: string,
  data: { title: string; level_order?: number; description?: string; requirements?: string },
): Promise<CareerLevel> {
  const res = await put<{ data: CareerLevel }>(`/career/levels/${id}`, data)
  return res.data
}

export async function deleteCareerLevel(id: string): Promise<void> {
  await del(`/career/levels/${id}`)
}

export async function listCareerPaths(): Promise<CareerPath[]> {
  const res = await get<{ data: CareerPath[] }>('/career/paths')
  return res.data
}

export async function upsertCareerPath(data: {
  employee_id: string
  current_level_id?: string
  target_level_id?: string
  target_date?: string
  notes?: string
}): Promise<CareerPath> {
  const res = await post<{ data: CareerPath }>('/career/paths', data)
  return res.data
}

export async function deleteCareerPath(id: string): Promise<void> {
  await del(`/career/paths/${id}`)
}
