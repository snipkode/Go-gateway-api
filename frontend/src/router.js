import { createRouter, createWebHistory } from 'vue-router'
import { auth } from './lib/api.js'

const routes = [
  { path: '/login', name: 'login', component: () => import('./views/LoginView.vue'), meta: { guest: true } },
  { path: '/', name: 'dashboard', component: () => import('./views/DashboardView.vue'), props: true },
  { path: '/apis', name: 'apis', component: () => import('./views/MicroHost.vue'), props: () => ({ name: 'registry' }) },
  { path: '/apis/:id/edit', name: 'api-edit', component: () => import('./views/MicroHost.vue'), props: (r) => ({ name: 'registry', init: r.params.id }) },
  { path: '/simulate', name: 'simulate', component: () => import('./views/MicroHost.vue'), props: () => ({ name: 'simulator' }) },
  { path: '/monitoring', name: 'monitoring', component: () => import('./views/MicroHost.vue'), props: () => ({ name: 'monitoring' }) },
  { path: '/docs', name: 'docs', component: () => import('./views/MicroHost.vue'), props: () => ({ name: 'docs' }) },
  { path: '/:pathMatch(.*)*', redirect: '/' }
]

export const router = createRouter({
  history: createWebHistory('/admin/'),
  routes
})

router.beforeEach((to) => {
  if (!to.meta?.guest && !auth.token) {
    return { name: 'login', query: to.fullPath !== '/' ? { next: to.fullPath } : {} }
  }
  if (to.name === 'login' && auth.token) return { name: 'dashboard' }
  return true
})