import { get, post, put, del } from './client'
import type { Project } from '@/types'

export async function listProjects(): Promise<Project[]> {
  const res = await get<{ data: Project[] }>('/projects')
  return res.data
}

export async function getProject(id: string): Promise<Project> {
  const res = await get<{ data: Project }>(`/projects/${id}`)
  return res.data
}

export async function createProject(data: {
  name: string
  description?: string
  owner_id?: string
  owner_team_id?: string
  status?: string
  priority?: string
  linked_goal_ids?: string[]
  linked_metric_ids?: string[]
  blockers?: string[]
  start_date?: string
  due_date?: string
}): Promise<Project> {
  const res = await post<{ data: Project }>('/projects', data)
  return res.data
}

export async function updateProject(
  id: string,
  data: {
    name: string
    description?: string
    owner_id?: string
    owner_team_id?: string
    status?: string
    priority?: string
    linked_goal_ids?: string[]
    linked_metric_ids?: string[]
    blockers?: string[]
    start_date?: string
    due_date?: string
  },
): Promise<Project> {
  const res = await put<{ data: Project }>(`/projects/${id}`, data)
  return res.data
}

export async function deleteProject(id: string): Promise<void> {
  await del(`/projects/${id}`)
}
