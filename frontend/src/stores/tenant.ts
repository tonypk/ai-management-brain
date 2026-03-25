import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Tenant } from '@/types'
import { getTenant } from '@/api/settings'

export const useTenantStore = defineStore('tenant', () => {
  const tenant = ref<Tenant | null>(null)
  const loading = ref(false)

  async function load(): Promise<Tenant> {
    if (tenant.value) return tenant.value
    loading.value = true
    try {
      tenant.value = await getTenant()
      return tenant.value
    } finally {
      loading.value = false
    }
  }

  function clear(): void {
    tenant.value = null
  }

  return { tenant, loading, load, clear }
})
