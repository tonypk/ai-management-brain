<script setup lang="ts">
import { h, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NLayout, NLayoutSider, NLayoutContent, NMenu, NButton, NIcon, type MenuOption } from 'naive-ui'
import {
  GridOutline as DashboardIcon,
  AlertCircleOutline as AlertIcon,
  DocumentTextOutline as ReportIcon,
  SettingsOutline as SettingsIcon,
  LogOutOutline as LogoutIcon,
} from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const themeStore = useThemeStore()

const activeKey = computed(() => {
  const path = route.path
  if (path === '/') return 'dashboard'
  return path.slice(1)
})

function renderIcon(icon: typeof DashboardIcon) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions: MenuOption[] = [
  { label: 'Dashboard', key: 'dashboard', icon: renderIcon(DashboardIcon) },
  { label: 'Alerts', key: 'alerts', icon: renderIcon(AlertIcon) },
  { label: 'Reports', key: 'reports', icon: renderIcon(ReportIcon) },
  { label: 'Settings', key: 'settings', icon: renderIcon(SettingsIcon) },
]

function handleMenuUpdate(key: string): void {
  const path = key === 'dashboard' ? '/' : `/${key}`
  router.push(path)
}

function handleLogout(): void {
  authStore.logout()
  router.push('/login')
}

const isMobile = computed(() => typeof window !== 'undefined' && window.innerWidth < 768)
</script>

<template>
  <NLayout has-sider style="min-height: 100vh">
    <NLayoutSider
      bordered
      :collapsed="themeStore.sidebarCollapsed"
      :collapsed-width="64"
      :width="220"
      collapse-mode="width"
      show-trigger="bar"
      :native-scrollbar="false"
      inverted
      :position="isMobile ? 'absolute' : 'static'"
      @collapse="themeStore.sidebarCollapsed = true"
      @expand="themeStore.sidebarCollapsed = false"
      style="background: #1a1a2e"
    >
      <div style="padding: 20px 16px 16px; border-bottom: 1px solid #2d2d44">
        <div v-if="!themeStore.sidebarCollapsed" style="color: #fff; font-size: 18px; font-weight: 700">
          AI Brain
        </div>
        <div v-if="!themeStore.sidebarCollapsed" style="color: #888; font-size: 12px; margin-top: 2px">
          Management Dashboard
        </div>
        <div v-else style="color: #fff; font-size: 18px; font-weight: 700; text-align: center">
          AI
        </div>
      </div>

      <NMenu
        :value="activeKey"
        :options="menuOptions"
        :collapsed="themeStore.sidebarCollapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        inverted
        style="padding-top: 8px"
        @update:value="handleMenuUpdate"
      />

      <div style="padding: 12px 16px; border-top: 1px solid #2d2d44; margin-top: auto">
        <NButton
          text
          style="color: #888; width: 100%"
          @click="handleLogout"
        >
          <template #icon>
            <NIcon :component="LogoutIcon" />
          </template>
          <span v-if="!themeStore.sidebarCollapsed">Logout</span>
        </NButton>
      </div>
    </NLayoutSider>

    <NLayoutContent style="padding: 24px; background: #f5f7fa">
      <slot />
    </NLayoutContent>
  </NLayout>
</template>
