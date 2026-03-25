import type { NavigationGuardNext, RouteLocationNormalized } from 'vue-router'
import { hasToken } from '@/api/client'

export function authGuard(
  to: RouteLocationNormalized,
  _from: RouteLocationNormalized,
  next: NavigationGuardNext,
): void {
  if (to.meta.requiresAuth && !hasToken()) {
    if (to.path === '/') {
      next({ name: 'Landing' })
    } else {
      next({ name: 'Login', query: { redirect: to.fullPath } })
    }
  } else {
    next()
  }
}
