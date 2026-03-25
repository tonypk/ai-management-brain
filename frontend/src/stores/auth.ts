import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { hasToken, clearToken } from '@/api/client'
import { login as apiLogin, register as apiRegister, googleAuth as apiGoogleAuth } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const isAuthenticated = computed(() => !!token.value && hasToken())

  async function login(email: string, password: string): Promise<void> {
    const t = await apiLogin(email, password)
    token.value = t
  }

  async function register(email: string, password: string, tenantName: string): Promise<void> {
    const t = await apiRegister(email, password, tenantName)
    token.value = t
  }

  async function googleLogin(credential: string): Promise<void> {
    const t = await apiGoogleAuth(credential)
    token.value = t
  }

  function logout(): void {
    clearToken()
    token.value = ''
  }

  return { token, isAuthenticated, login, register, googleLogin, logout }
})
