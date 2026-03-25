import { post, get, del, setToken } from './client'
import type { AuthResponse, ApiKey, ApiKeyCreateResponse } from '@/types'

export async function login(email: string, password: string): Promise<string> {
  const res = await post<AuthResponse>('/auth/login', { email, password })
  setToken(res.token)
  return res.token
}

export async function register(email: string, password: string, tenantName: string): Promise<string> {
  const res = await post<AuthResponse>('/auth/register', { email, password, tenant_name: tenantName })
  setToken(res.token)
  return res.token
}

export async function googleAuth(credential: string): Promise<string> {
  const res = await post<AuthResponse>('/auth/google', { credential })
  setToken(res.token)
  return res.token
}

export async function getGoogleClientId(): Promise<string> {
  const res = await get<{ client_id: string }>('/auth/google/client-id')
  return res.client_id
}

export async function createApiKey(name: string): Promise<ApiKeyCreateResponse> {
  const res = await post<{ data: ApiKeyCreateResponse }>('/auth/api-keys', { name })
  return res.data
}

export async function listApiKeys(): Promise<ApiKey[]> {
  const res = await get<{ data: ApiKey[] }>('/auth/api-keys')
  return res.data
}

export async function revokeApiKey(id: string): Promise<void> {
  await del<{ data: { deleted: boolean } }>(`/auth/api-keys/${id}`)
}
