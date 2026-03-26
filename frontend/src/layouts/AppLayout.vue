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
  PeopleOutline as PeopleIcon,
  SchoolOutline as MentorIcon,
  BusinessOutline as SeatsIcon,
  GitNetworkOutline as OrgIcon,
  MapOutline as SentimentIcon,
  ChatbubblesOutline as CoachingIcon,
  ClipboardOutline as BoardRecordsIcon,
  RibbonOutline as GoalsIcon,
  BulbOutline as InsightsIcon,
  NewspaperOutline as DigestIcon,
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
  if (path.startsWith('/employees/')) return 'sentiment'
  return path.slice(1)
})

function renderIcon(icon: typeof DashboardIcon) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions: MenuOption[] = [
  { type: 'group', label: 'Observe', key: 'observe-group', children: [
    { label: 'Dashboard', key: 'dashboard', icon: renderIcon(DashboardIcon) },
    { label: 'Alerts', key: 'alerts', icon: renderIcon(AlertIcon) },
    { label: 'Reports', key: 'reports', icon: renderIcon(ReportIcon) },
  ]},
  { type: 'group', label: 'Organize', key: 'organize-group', children: [
    { label: 'Team Members', key: 'employees', icon: renderIcon(PeopleIcon) },
    { label: 'Organization', key: 'organization', icon: renderIcon(OrgIcon) },
    { label: 'Mentor', key: 'mentor', icon: renderIcon(MentorIcon) },
    { label: 'C-Suite Board', key: 'seats', icon: renderIcon(SeatsIcon) },
  ]},
  { type: 'group', label: 'Lead', key: 'lead-group', children: [
    { label: 'Sentiment Map', key: 'sentiment', icon: renderIcon(SentimentIcon) },
    { label: '1:1 Coaching', key: 'coaching', icon: renderIcon(CoachingIcon) },
  ]},
  { type: 'group', label: 'Plan', key: 'plan-group', children: [
    { label: 'Board Records', key: 'board-records', icon: renderIcon(BoardRecordsIcon) },
    { label: 'Goals & KPIs', key: 'goals', icon: renderIcon(GoalsIcon) },
  ]},
  { type: 'group', label: 'Analyze', key: 'analyze-group', children: [
    { label: 'AI Insights', key: 'insights', icon: renderIcon(InsightsIcon) },
    { label: 'Weekly Digest', key: 'digest', icon: renderIcon(DigestIcon) },
  ]},
  { type: 'group', label: 'Configure', key: 'configure-group', children: [
    { label: 'Settings', key: 'settings', icon: renderIcon(SettingsIcon) },
  ]},
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
