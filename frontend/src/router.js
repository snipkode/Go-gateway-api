import { createRouter, createWebHistory } from 'vue-router'
import { auth } from './lib/api.js'

// In dev the micro sources are imported directly into the shell app so they
// share a single Vue instance (HMR + reactive updates). In production the
// shell composes them at runtime via MicroHost + mf-manifest.json.
const isDev = import.meta.env.DEV

const micro = (name) => {
  if (isDev) {
    const views = {
      registry: () => import('./micro/registry/RegistryApp.vue'),
      simulator: () => import('./micro/simulator/SimulatorApp.vue'),
      monitoring: () => import('./micro/monitoring/MonitoringApp.vue'),
      docs: () => import('./micro/docs/DocsApp.vue')
    }
    return views[name]
  }
  return () => import('./views/MicroHost.vue')
}

const routes = [
  { path: '/login', name: 'login', component: () => import('./views/LoginView.vue'), meta: { guest: true } },
  { path: '/', name: 'dashboard', component: () => import('./views/DashboardView.vue'), props: true },
  { path: '/apis', name: 'apis', component: micro('registry'), props: () => ({ name: 'registry', init: '' }) },
  { path: '/apis/:id/edit', name: 'api-edit', component: micro('registry'), props: (r) => ({ name: 'registry', init: r.params.id }) },
  { path: '/simulate', name: 'simulate', component: micro('simulator'), props: () => ({ name: 'simulator' }) },
  { path: '/monitoring', name: 'monitoring', component: micro('monitoring'), props: () => ({ name: 'monitoring' }) },
  { path: '/docs', name: 'docs', component: micro('docs'), props: () => ({ name: 'docs' }) },
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