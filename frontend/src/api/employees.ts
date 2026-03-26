import { get, post, put, del } from './client'
import type { Employee, EmployeeWithChannels } from '@/types'

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

export async function getEmployee(id: string): Promise<Employee> {
  const res = await get<{ data: Employee }>(`/employees/${id}`)
  return res.data
}

export async function updateEmployee(
  id: string,
  data: { name?: string; culture_code?: string; job_title?: string; responsibilities?: string; country?: string; language?: string },
): Promise<void> {
  await put<{ data: unknown }>(`/employees/${id}`, data)
}

export async function deleteEmployee(id: string): Promise<void> {
  await del<{ data: unknown }>(`/employees/${id}`)
}

export async function getEmployeeChannels(id: string): Promise<EmployeeWithChannels> {
  const res = await get<{ data: EmployeeWithChannels }>(`/employees/${id}/channels`)
  return res.data
}

export async function updateEmployeeChannels(
  id: string,
  data: { telegram_id?: string; slack_id?: string; lark_id?: string; preferred_channel?: string },
): Promise<void> {
  await put<{ data: unknown }>(`/employees/${id}/channels`, data)
}
