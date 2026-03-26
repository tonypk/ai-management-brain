import { get, post, put, del } from './client'
import type { Task, TaskStats } from '@/types'

export async function listTasks(): Promise<Task[]> {
  const res = await get<{ data: Task[] }>('/tasks')
  return res.data
}

export async function getTask(id: string): Promise<Task> {
  const res = await get<{ data: Task }>(`/tasks/${id}`)
  return res.data
}

export async function createTask(data: {
  title: string
  description?: string
  project_id?: string
  goal_id?: string
  key_result_id?: string
  owner_id?: string
  owner_team_id?: string
  status?: string
  priority?: string
  due_at?: string
}): Promise<Task> {
  const res = await post<{ data: Task }>('/tasks', data)
  return res.data
}

export async function updateTask(
  id: string,
  data: {
    title: string
    description?: string
    project_id?: string
    goal_id?: string
    key_result_id?: string
    owner_id?: string
    owner_team_id?: string
    status?: string
    priority?: string
    due_at?: string
  },
): Promise<Task> {
  const res = await put<{ data: Task }>(`/tasks/${id}`, data)
  return res.data
}

export async function deleteTask(id: string): Promise<void> {
  await del(`/tasks/${id}`)
}

export async function listOverdueTasks(): Promise<Task[]> {
  const res = await get<{ data: Task[] }>('/tasks/overdue')
  return res.data
}

export async function getTaskStats(): Promise<TaskStats[]> {
  const res = await get<{ data: TaskStats[] }>('/tasks/stats')
  return res.data
}
