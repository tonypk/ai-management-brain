import { get, post, put, del } from './client'
import type { Skill, EmployeeSkill, SkillMatrixEntry } from '@/types'

export async function listSkills(): Promise<Skill[]> {
  const res = await get<{ data: Skill[] }>('/skills')
  return res.data
}

export async function createSkill(data: {
  name: string
  category?: string
  description?: string
}): Promise<Skill> {
  const res = await post<{ data: Skill }>('/skills', data)
  return res.data
}

export async function updateSkill(id: string, data: {
  name: string
  category?: string
  description?: string
}): Promise<void> {
  await put<{ data: unknown }>(`/skills/${id}`, data)
}

export async function deleteSkill(id: string): Promise<void> {
  await del<{ data: unknown }>(`/skills/${id}`)
}

export async function getSkillMatrix(): Promise<SkillMatrixEntry[]> {
  const res = await get<{ data: SkillMatrixEntry[] }>('/skills/matrix')
  return res.data
}

export async function listEmployeeSkills(employeeId: string): Promise<EmployeeSkill[]> {
  const res = await get<{ data: EmployeeSkill[] }>(`/skills/employees/${employeeId}`)
  return res.data
}

export async function setEmployeeSkill(employeeId: string, data: {
  skill_id: string
  level: number
  notes?: string
}): Promise<EmployeeSkill> {
  const res = await post<{ data: EmployeeSkill }>(`/skills/employees/${employeeId}`, data)
  return res.data
}

export async function deleteEmployeeSkill(employeeId: string, skillId: string): Promise<void> {
  await del<{ data: unknown }>(`/skills/employees/${employeeId}/${skillId}`)
}
