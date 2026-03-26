import { get, post, put, del } from './client'
import type { Meeting, ActionItem, OpenActionItem } from '@/types'

export async function listMeetings(limit = 50, offset = 0): Promise<Meeting[]> {
  const res = await get<{ data: Meeting[] }>(`/meetings?limit=${limit}&offset=${offset}`)
  return res.data
}

export async function getMeeting(id: string): Promise<Meeting> {
  const res = await get<{ data: Meeting }>(`/meetings/${id}`)
  return res.data
}

export async function createMeeting(data: {
  employee_id: string
  manager_id?: string
  meeting_date: string
  duration_min?: number
  notes?: string
  mood?: string
  follow_up?: string
}): Promise<Meeting> {
  const res = await post<{ data: Meeting }>('/meetings', data)
  return res.data
}

export async function updateMeeting(id: string, data: {
  notes: string
  mood: string
  follow_up: string
  duration_min: number
}): Promise<void> {
  await put<{ data: unknown }>(`/meetings/${id}`, data)
}

export async function deleteMeeting(id: string): Promise<void> {
  await del<{ data: unknown }>(`/meetings/${id}`)
}

export async function listActionItems(meetingId: string): Promise<ActionItem[]> {
  const res = await get<{ data: ActionItem[] }>(`/meetings/${meetingId}/actions`)
  return res.data
}

export async function createActionItem(meetingId: string, data: {
  title: string
  assignee_id?: string | null
  due_date?: string | null
}): Promise<ActionItem> {
  const res = await post<{ data: ActionItem }>(`/meetings/${meetingId}/actions`, data)
  return res.data
}

export async function updateActionItem(meetingId: string, itemId: string, data: {
  title: string
  status: string
  assignee_id?: string | null
  due_date?: string | null
}): Promise<void> {
  await put<{ data: unknown }>(`/meetings/${meetingId}/actions/${itemId}`, data)
}

export async function deleteActionItem(meetingId: string, itemId: string): Promise<void> {
  await del<{ data: unknown }>(`/meetings/${meetingId}/actions/${itemId}`)
}

export async function listOpenActionItems(): Promise<OpenActionItem[]> {
  const res = await get<{ data: OpenActionItem[] }>('/meetings/actions/open')
  return res.data
}
