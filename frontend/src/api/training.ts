import { get, post, put, del } from './client'
import type { TrainingProgram, TrainingEnrollment } from '@/types'

export async function listTrainingPrograms(): Promise<TrainingProgram[]> {
  const res = await get<{ data: TrainingProgram[] }>('/training')
  return res.data
}

export async function createTrainingProgram(data: {
  title: string
  description?: string
  category?: string
  duration_hours?: number
  provider?: string
  url?: string
  is_mandatory?: boolean
}): Promise<TrainingProgram> {
  const res = await post<{ data: TrainingProgram }>('/training', data)
  return res.data
}

export async function updateTrainingProgram(
  id: string,
  data: {
    title: string
    description?: string
    category?: string
    duration_hours?: number
    provider?: string
    url?: string
    is_mandatory?: boolean
    status?: string
  },
): Promise<TrainingProgram> {
  const res = await put<{ data: TrainingProgram }>(`/training/${id}`, data)
  return res.data
}

export async function deleteTrainingProgram(id: string): Promise<void> {
  await del(`/training/${id}`)
}

export async function listEnrollments(programId: string): Promise<TrainingEnrollment[]> {
  const res = await get<{ data: TrainingEnrollment[] }>(`/training/${programId}/enrollments`)
  return res.data
}

export async function createEnrollment(
  programId: string,
  data: { employee_id: string },
): Promise<TrainingEnrollment> {
  const res = await post<{ data: TrainingEnrollment }>(`/training/${programId}/enrollments`, data)
  return res.data
}

export async function updateEnrollment(
  programId: string,
  enrollmentId: string,
  data: { status: string; score?: number | null; notes?: string },
): Promise<TrainingEnrollment> {
  const res = await put<{ data: TrainingEnrollment }>(
    `/training/${programId}/enrollments/${enrollmentId}`,
    data,
  )
  return res.data
}

export async function deleteEnrollment(programId: string, enrollmentId: string): Promise<void> {
  await del(`/training/${programId}/enrollments/${enrollmentId}`)
}
