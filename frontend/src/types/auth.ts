export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  password: string
  tenant_name: string
}

export interface AuthResponse {
  token: string
}

export interface ApiKey {
  id: string
  prefix: string
  name: string
  created_at: string
  last_used_at: string | null
}

export interface ApiKeyCreateResponse {
  id: string
  key: string
  prefix: string
  name: string
}
