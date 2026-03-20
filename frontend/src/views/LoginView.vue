<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login, register } from '../composables/api'

const router = useRouter()
const isRegister = ref(false)
const email = ref('')
const password = ref('')
const tenantName = ref('')
const error = ref('')
const loading = ref(false)

async function handleSubmit() {
  error.value = ''
  loading.value = true
  try {
    if (isRegister.value) {
      await register(email.value, password.value, tenantName.value)
    } else {
      await login(email.value, password.value)
    }
    router.push('/')
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <h1>AI Management Brain</h1>
      <p class="subtitle">{{ isRegister ? 'Create your team' : 'Sign in to your dashboard' }}</p>

      <form @submit.prevent="handleSubmit">
        <div class="field">
          <label>Email</label>
          <input v-model="email" type="email" placeholder="you@company.com" required />
        </div>
        <div class="field">
          <label>Password</label>
          <input v-model="password" type="password" placeholder="Min 8 characters" required minlength="8" />
        </div>
        <div v-if="isRegister" class="field">
          <label>Team Name</label>
          <input v-model="tenantName" type="text" placeholder="Your team or company" required />
        </div>

        <p v-if="error" class="error-msg">{{ error }}</p>

        <button type="submit" class="btn btn-primary submit-btn" :disabled="loading">
          {{ loading ? 'Please wait...' : (isRegister ? 'Create Account' : 'Sign In') }}
        </button>
      </form>

      <p class="toggle">
        {{ isRegister ? 'Already have an account?' : "Don't have an account?" }}
        <a href="#" @click.prevent="isRegister = !isRegister; error = ''">
          {{ isRegister ? 'Sign in' : 'Create one' }}
        </a>
      </p>
    </div>
  </div>
</template>

<style scoped>
.login-page { display: flex; align-items: center; justify-content: center; min-height: 100vh; background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%); }
.login-card { background: #fff; border-radius: 16px; padding: 2.5rem; width: 400px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.login-card h1 { font-size: 1.5rem; margin-bottom: 0.25rem; }
.subtitle { color: #888; margin-bottom: 2rem; font-size: 0.9rem; }
.field { margin-bottom: 1rem; }
.field label { display: block; font-size: 0.85rem; font-weight: 600; color: #555; margin-bottom: 0.25rem; }
.field input { width: 100%; }
.submit-btn { width: 100%; padding: 0.75rem; margin-top: 0.5rem; font-weight: 600; }
.toggle { text-align: center; margin-top: 1.5rem; font-size: 0.85rem; color: #888; }
.toggle a { color: #6366f1; text-decoration: none; }
</style>
