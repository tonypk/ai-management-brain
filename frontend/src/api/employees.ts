import { get, post, put } from './client'
import type { Employee } from '@/types'

export async function listEmployees(): Promise<Employee[]> {
  const res = await get<{ data: Employee[] }>('/employees')
  return res.data
}

export async function createEmployee(data: {
  name: string
  culture_code: string
  job_title?: string
  responsibilities?: string
  country?: string
  language?: string
}): Promise<Employee> {
  const res = await post<{ data: Employee }>('/employees', data)
  return res.data
}

export async function updateEmployeeProfile(
  id: string,
  data: { job_title?: string; responsibilities?: string; country?: string; language?: string },
): Promise<void> {
  await put<{ data: unknown }>(`/employees/${id}/profile`, data)
}
