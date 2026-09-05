<script setup>
import { useRouter } from 'vue-router'
import { auth } from './api'
import { useToast } from './toast'

const toast = useToast()
const router = useRouter()
const routes = [
  { to: '/dashboard', label: 'Dashboard', icon: '▦' },
  { to: '/apis', label: 'API Registry', icon: '∿' },
  { to: '/simulate', label: 'Configure & Simulate', icon: '⚙' },
  { to: '/monitoring', label: 'Monitoring', icon: '◔' },
  { to: '/docs', label: 'Gateway Docs', icon: '⌘' }
]

function logout() {
  auth.clear()
  router.push({ name: 'login' })
}
</script>

<template>
  <aside
    class="w-56 shrink-0 border-r border-line bg-panel px-3 py-4 flex flex-col gap-1 h-screen sticky top-0"
  >
    <div class="px-3 pb-4">
      <h1 class="text-[15px] font-semibold tracking-wide m-0">API Gateway</h1>
      <small class="text-mute">Management Console</small>
    </div>
    <RouterLink
      v-for="r in routes"
      :key="r.to"
      :to="r.to"
      class="nav-link flex items-center gap-2.5 rounded-lg px-3 py-2.5 text-mute no-underline transition-colors"
      active-class="!bg-accent !text-white"
    >
      <span>{{ r.icon }}</span> {{ r.label }}
    </RouterLink>
    <div v-if="auth.user" class="mt-auto border-t border-line px-3 py-3">
      <strong class="block text-ink">{{ auth.user.name }}</strong>
      <span class="text-mute text-xs">{{ auth.user.email }}</span>
      <button
        class="mt-2 w-full rounded-md border border-line px-2.5 py-1.5 text-mute cursor-pointer transition-colors hover:border-bad hover:text-bad"
        @click="logout"
      >
        Sign out
      </button>
    </div>
  </aside>
  <main class="flex-1 min-w-0 px-8 py-7 pb-16">
    <RouterView />
    <div
      v-if="toast.msg"
      class="fixed right-6 bottom-6 z-50 max-w-sm rounded-lg border px-4 py-3 shadow-xl"
      :class="toast.kind === 'ok' ? 'border-ok bg-panel-2' : 'border-bad bg-panel-2'"
    >
      <span :class="toast.kind === 'ok' ? 'text-ok' : 'text-bad'">{{ toast.msg }}</span>
    </div>
  </main>
</template>

<style>
.nav-link:hover {
  background: var(--color-panel-2);
  color: var(--color-ink);
}
</style>