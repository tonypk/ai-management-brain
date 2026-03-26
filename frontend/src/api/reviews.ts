import { get, post, put, del } from './client'
import type { ReviewCycle, PerformanceReview } from '@/types'

export async function listReviewCycles(): Promise<ReviewCycle[]> {
  const res = await get<{ data: ReviewCycle[] }>('/reviews/cycles')
  return res.data
}

export async function createReviewCycle(data: {
  title: string
  period: string
  start_date: string
  end_date: string
  status?: string
}): Promise<ReviewCycle> {
  const res = await post<{ data: ReviewCycle }>('/reviews/cycles', data)
  return res.data
}

export async function updateReviewCycle(id: string, data: {
  title: string
  status: string
  start_date: string
  end_date: string
}): Promise<void> {
  await put<{ data: unknown }>(`/reviews/cycles/${id}`, data)
}

export async function deleteReviewCycle(id: string): Promise<void> {
  await del<{ data: unknown }>(`/reviews/cycles/${id}`)
}

export async function listReviews(cycleId: string): Promise<PerformanceReview[]> {
  const res = await get<{ data: PerformanceReview[] }>(`/reviews/cycles/${cycleId}/reviews`)
  return res.data
}

export async function createReview(cycleId: string, data: {
  employee_id: string
  reviewer_id?: string | null
}): Promise<PerformanceReview> {
  const res = await post<{ data: PerformanceReview }>(`/reviews/cycles/${cycleId}/reviews`, data)
  return res.data
}

export async function updateReview(cycleId: string, reviewId: string, data: {
  status: string
  self_rating?: number | null
  manager_rating?: number | null
  self_summary?: string
  manager_summary?: string
  strengths?: string
  improvements?: string
}): Promise<void> {
  await put<{ data: unknown }>(`/reviews/cycles/${cycleId}/reviews/${reviewId}`, data)
}
