import { createRouter, createWebHashHistory } from "vue-router";
import { isAuthenticated } from "../composables/api";

const routes = [
  {
    path: "/landing",
    name: "Landing",
    component: () => import("../views/LandingView.vue"),
  },
  {
    path: "/login",
    name: "Login",
    component: () => import("../views/LoginView.vue"),
  },
  {
    path: "/",
    name: "Dashboard",
    component: () => import("../views/DashboardView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/employees",
    name: "Employees",
    component: () => import("../views/EmployeesView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/reports",
    name: "Reports",
    component: () => import("../views/ReportsView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/mentor",
    name: "Mentor",
    component: () => import("../views/MentorView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/organization",
    name: "Organization",
    component: () => import("../views/OrganizationView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/ai-roles",
    name: "AIRoles",
    component: () => import("../views/AIRolesView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/admin/channels",
    name: "AdminChannels",
    component: () => import("../views/admin/ChannelsView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/admin/team-channels",
    name: "AdminTeamChannels",
    component: () => import("../views/admin/TeamChannelsView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/admin/reports",
    name: "AdminReports",
    component: () => import("../views/admin/ReportsView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/admin/mentor-scheduler",
    name: "AdminMentorScheduler",
    component: () => import("../views/admin/MentorSchedulerView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/admin/memory",
    name: "AdminMemory",
    component: () => import("../views/admin/MemoryView.vue"),
    meta: { requiresAuth: true },
  },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

router.beforeEach((to) => {
  if (to.meta.requiresAuth && !isAuthenticated()) {
    return { name: "Login" };
  }
});

export default router;
