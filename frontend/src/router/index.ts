import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { authGuard } from './guards'

const routes: RouteRecordRaw[] = [
  {
    path: '/landing',
    name: 'Landing',
    component: () => import('@/views/LandingView.vue'),
    meta: { layout: 'landing' },
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/LoginView.vue'),
    meta: { layout: 'auth' },
  },
  {
    path: '/',
    meta: { requiresAuth: true, layout: 'app' },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: () => import('@/views/DashboardView.vue'),
      },
      {
        path: 'alerts',
        name: 'Alerts',
        component: () => import('@/views/AlertsView.vue'),
      },
      {
        path: 'reports',
        name: 'Reports',
        component: () => import('@/views/ReportsView.vue'),
      },
      {
        path: 'employees',
        name: 'Employees',
        component: () => import('@/views/EmployeesView.vue'),
      },
      {
        path: 'organization',
        name: 'Organization',
        component: () => import('@/views/OrganizationView.vue'),
      },
      {
        path: 'mentor',
        name: 'Mentor',
        component: () => import('@/views/MentorView.vue'),
      },
      {
        path: 'seats',
        name: 'Seats',
        component: () => import('@/views/SeatsView.vue'),
      },
      {
        path: 'sentiment',
        name: 'Sentiment',
        component: () => import('@/views/SentimentMapView.vue'),
      },
      {
        path: 'employees/:id',
        name: 'EmployeeProfile',
        component: () => import('@/views/EmployeeProfileView.vue'),
      },
      {
        path: 'coaching',
        name: 'Coaching',
        component: () => import('@/views/CoachingView.vue'),
      },
      {
        path: 'board-records',
        name: 'BoardRecords',
        component: () => import('@/views/BoardRecordsView.vue'),
      },
      {
        path: 'goals',
        name: 'Goals',
        component: () => import('@/views/GoalsView.vue'),
      },
      {
        path: 'insights',
        name: 'Insights',
        component: () => import('@/views/InsightsView.vue'),
      },
      {
        path: 'digest',
        name: 'Digest',
        component: () => import('@/views/DigestView.vue'),
      },
      {
        path: 'reviews',
        name: 'Reviews',
        component: () => import('@/views/ReviewsView.vue'),
      },
      {
        path: 'meetings',
        name: 'Meetings',
        component: () => import('@/views/MeetingsView.vue'),
      },
      {
        path: 'skills',
        name: 'Skills',
        component: () => import('@/views/SkillsView.vue'),
      },
      {
        path: 'training',
        name: 'Training',
        component: () => import('@/views/TrainingView.vue'),
      },
      {
        path: 'career',
        name: 'Career',
        component: () => import('@/views/CareerView.vue'),
      },
      {
        path: 'metrics',
        name: 'Metrics',
        component: () => import('@/views/MetricsView.vue'),
      },
      {
        path: 'projects',
        name: 'Projects',
        component: () => import('@/views/ProjectsView.vue'),
      },
      {
        path: 'tasks',
        name: 'Tasks',
        component: () => import('@/views/TasksView.vue'),
      },
      {
        path: 'incentives',
        name: 'Incentives',
        component: () => import('@/views/IncentivesView.vue'),
      },
      {
        path: 'state',
        name: 'State',
        component: () => import('@/views/StateView.vue'),
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/views/SettingsView.vue'),
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/',
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(authGuard)

export default router
