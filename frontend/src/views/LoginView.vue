<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NTabs, NTabPane, NAlert, NDivider } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import { getGoogleClientId } from '@/api/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const mode = ref<'login' | 'register'>('login')
const email = ref('')
const password = ref('')
const tenantName = ref('')
const error = ref('')
const loading = ref(false)
const googleClientId = ref('')

onMounted(async () => {
  if (authStore.isAuthenticated) {
    router.push('/')
    return
  }
  try {
    googleClientId.value = await getGoogleClientId()
    if (googleClientId.value) {
      loadGoogleScript()
    }
  } catch {
    // Google OAuth not configured, skip
  }
})

function loadGoogleScript(): void {
  if (document.getElementById('google-gsi')) return
  const script = document.createElement('script')
  script.id = 'google-gsi'
  script.src = 'https://accounts.google.com/gsi/client'
  script.async = true
  script.defer = true
  script.onload = initGoogle
  document.head.appendChild(script)
}

function initGoogle(): void {
  const w = window as unknown as Record<string, unknown>
  const google = w.google as { accounts?: { id?: { initialize: (c: unknown) => void; renderButton: (e: HTMLElement, c: unknown) => void } } } | undefined
  if (!google?.accounts?.id) return
  google.accounts.id.initialize({
    client_id: googleClientId.value,
    callback: handleGoogleCallback,
  })
  const el = document.getElementById('google-btn')
  if (el) {
    google.accounts.id.renderButton(el, { theme: 'outline', size: 'large', width: 370 })
  }
}

async function handleGoogleCallback(response: { credential: string }): Promise<void> {
  error.value = ''
  loading.value = true
  try {
    await authStore.googleLogin(response.credential)
    navigateAfterAuth()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Google login failed'
  } finally {
    loading.value = false
  }
}

async function handleSubmit(): Promise<void> {
  error.value = ''
  loading.value = true
  try {
    if (mode.value === 'login') {
      await authStore.login(email.value, password.value)
    } else {
      await authStore.register(email.value, password.value, tenantName.value)
    }
    navigateAfterAuth()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Authentication failed'
  } finally {
    loading.value = false
  }
}

function navigateAfterAuth(): void {
  const redirect = route.query.redirect as string
  router.push(redirect || '/')
}
</script>

<template>
  <NCard style="border-radius: 12px">
    <div style="text-align: center; margin-bottom: 24px">
      <h1 style="font-size: 24px; font-weight: 700; color: #1a1a2e">AI Management Brain</h1>
      <p style="color: #888; font-size: 14px">Your AI-powered management OS</p>
    </div>

    <NAlert v-if="error" type="error" closable style="margin-bottom: 16px" @close="error = ''">
      {{ error }}
    </NAlert>

    <NTabs :value="mode" @update:value="(v: string) => mode = v as 'login' | 'register'" type="segment" animated>
      <NTabPane name="login" tab="Login">
        <NForm @submit.prevent="handleSubmit" style="margin-top: 16px">
          <NFormItem label="Email">
            <NInput v-model:value="email" type="text" placeholder="email@example.com" />
          </NFormItem>
          <NFormItem label="Password">
            <NInput v-model:value="password" type="password" show-password-on="click" placeholder="Password" />
          </NFormItem>
          <NButton type="primary" block :loading="loading" attr-type="submit">
            Login
          </NButton>
        </NForm>
      </NTabPane>

      <NTabPane name="register" tab="Register">
        <NForm @submit.prevent="handleSubmit" style="margin-top: 16px">
          <NFormItem label="Organization Name">
            <NInput v-model:value="tenantName" placeholder="My Company" />
          </NFormItem>
          <NFormItem label="Email">
            <NInput v-model:value="email" type="text" placeholder="email@example.com" />
          </NFormItem>
          <NFormItem label="Password">
            <NInput v-model:value="password" type="password" show-password-on="click" placeholder="Min 8 characters" />
          </NFormItem>
          <NButton type="primary" block :loading="loading" attr-type="submit">
            Register
          </NButton>
        </NForm>
      </NTabPane>
    </NTabs>

    <template v-if="googleClientId">
      <NDivider>or</NDivider>
      <div id="google-btn" style="display: flex; justify-content: center"></div>
    </template>
  </NCard>
</template>
