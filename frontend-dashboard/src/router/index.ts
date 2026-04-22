import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('@/components/layout/MarketingLayout.vue'),
    children: [{ path: '', component: () => import('@/views/HomeView.vue') }]
  },
  {
    path: '/',
    component: () => import('@/components/layout/AuthLayout.vue'),
    children: [
      { path: 'login', component: () => import('@/views/LoginView.vue') },
      { path: 'register', component: () => import('@/views/RegisterView.vue') }
    ]
  },
  {
    path: '/',
    component: () => import('@/components/layout/AppLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: 'dashboard', component: () => import('@/views/DashboardView.vue') },
      { path: 'keys', component: () => import('@/views/KeysView.vue') },
      { path: 'usage', component: () => import('@/views/UsageView.vue') },
      { path: 'groups', component: () => import('@/views/GroupsView.vue') },
      { path: 'profile', component: () => import('@/views/ProfileView.vue') }
    ]
  },
  { path: '/:pathMatch(.*)*', component: () => import('@/views/NotFoundView.vue') }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to) {
    if (to.hash) return { el: to.hash, behavior: 'smooth' }
    return { top: 0 }
  }
})

router.beforeEach((to) => {
  if (to.meta?.requiresAuth) {
    const auth = useAuthStore()
    if (!auth.isAuthed) return { path: '/login', query: { redirect: to.fullPath } }
  }
  return true
})

export default router
