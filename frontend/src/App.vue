<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NConfigProvider, NMessageProvider, NDialogProvider, type GlobalThemeOverrides } from 'naive-ui'
import AppLayout from '@/layouts/AppLayout.vue'
import AuthLayout from '@/layouts/AuthLayout.vue'
import LandingLayout from '@/layouts/LandingLayout.vue'

const route = useRoute()
const router = useRouter()
const routerReady = ref(false)
router.isReady().then(() => { routerReady.value = true })

const layoutName = computed(() => {
  return (route.meta.layout as string) || 'app'
})

const layoutComponent = computed(() => {
  switch (layoutName.value) {
    case 'auth': return AuthLayout
    case 'landing': return LandingLayout
    default: return AppLayout
  }
})

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#6366f1',
    primaryColorHover: '#818cf8',
    primaryColorPressed: '#4f46e5',
    primaryColorSuppl: '#6366f1',
  },
}
</script>

<template>
  <NConfigProvider :theme-overrides="themeOverrides">
    <NMessageProvider>
      <NDialogProvider>
        <component v-if="routerReady" :is="layoutComponent">
          <router-view />
        </component>
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>

<style>
body {
  margin: 0;
  padding: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: #f5f7fa;
  color: #1a1a2e;
}
</style>
