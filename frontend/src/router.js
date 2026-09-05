import { createRouter, createWebHistory } from 'vue-router'
import { auth } from './api'

import LoginView from './views/LoginView.vue'
import DashboardView from './views/DashboardView.vue'
import ApisView from './views/ApisView.vue'
import ApiFormView from './views/ApiFormView.vue'
import SimulatorView from './views/SimulatorView.vue'
import MonitoringView from './views/MonitoringView.vue'
import DocsView from './views/DocsView.vue'

const router = createRouter({
  history: createWebHistory('/admin/'),
  routes: [
    { path: '/login', name: 'login', component: LoginView, meta: { public: true } },
    { path: '/', redirect: '/dashboard' },
    { path: '/dashboard', name: 'dashboard', component: DashboardView },
    { path: '/apis', name: 'apis', component: ApisView },
    { path: '/apis/new', name: 'api-new', component: ApiFormView },
    { path: '/apis/:id/edit', name: 'api-edit', component: ApiFormView },
    { path: '/simulate', name: 'simulate', component: SimulatorView },
    { path: '/monitoring', name: 'monitoring', component: MonitoringView },
    { path: '/docs', name: 'docs', component: DocsView }
  ]
})

router.beforeEach((to) => {
  const loggedIn = !!auth.token
  if (!to.meta.public && !loggedIn) return { name: 'login' }
  if (to.name === 'login' && loggedIn) return { name: 'dashboard' }
  return true
})

export default router