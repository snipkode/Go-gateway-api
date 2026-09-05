<script setup>
import { ref, computed, onMounted, provide } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { auth } from './lib/api.js'
import { useToast } from './lib/toast.js'

const toast = useToast()
const router = useRouter()
const route = useRoute()
const mfManifest = ref([])
provide('mfManifest', mfManifest)

const nav = [
  { name: 'dashboard', label: 'Dashboard', to: '/', icon: 'M4 5h6v6H4zM14 5h6v6h-6zM4 15h6v6H4zM14 15h6v6h-6z' },
  { name: 'apis', label: 'Registry', to: '/apis', icon: 'M4 4h16v5H4zM4 11h16v5H4zM4 18h16v2H4z' },
  { name: 'simulate', label: 'Simulate', to: '/simulate', icon: 'M7 9h10M7 13h10M5 17h14' },
  { name: 'monitoring', label: 'Monitor', to: '/monitoring', icon: 'M5 19V9M10 19V5M15 19v-7M20 19V3' },
  { name: 'docs', label: 'Docs', to: '/docs', icon: 'M5 4h14v16H5zM9 8h6M9 12h6' }
]

const page = computed(() => nav.find((n) => route.path.startsWith(n.to)) || nav[0])
const user = computed(() => auth.user || {})

onMounted(async () => {
  try {
    const res = await fetch('/admin/mf-manifest.json')
    mfManifest.value = await res.json()
  } catch {
    mfManifest.value = []
  }
})

function logout() {
  auth.clear()
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="min-h-screen">
    <!-- desktop sidebar -->
    <aside class="fixed left-0 top-0 z-30 hidden h-screen w-60 flex-col bg-panel/80 py-5 pl-5 md:flex md:border-r" style="border-color: var(--color-line); backdrop-filter: blur(20px)">
      <div class="mb-8 flex items-center gap-2.5 px-1">
        <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-accent shadow-sm">
          <svg width="18" height="18" viewBox="0 0 24 24" class="text-white"><path fill="none" stroke="currentColor" stroke-width="2" d="M4 6h16v4H4zM4 12h16v4H4zM4 18h10"/></svg>
        </div>
        <div>
          <div class="text-[14px] font-bold leading-tight">Gateway Console</div>
          <div class="text-[11px] text-mute">go-enterprise-api</div>
        </div>
      </div>

      <nav class="flex-1 space-y-0.5">
        <RouterLink
          v-for="n in nav"
          :key="n.name"
          :to="n.to"
          class="tappable flex items-center gap-3 rounded-xl px-3 py-2 text-[13px] font-medium transition-colors"
          :class="route.path.startsWith(n.to) ? 'bg-accent-tint text-accent-2' : 'text-mute hover:text-ink'"
        >
          <svg width="17" height="17" viewBox="0 0 24 24" fill="none" class="shrink-0">
            <path :d="n.icon" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" stroke-linecap="round" />
            <path v-if="n.name === 'simulate'" :d="'M7 9h10M7 13h10'" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
          </svg>
          {{ n.label }}
        </RouterLink>
      </nav>

      <div class="mt-4 flex items-center gap-3 rounded-2xl bg-panel-2 px-3 py-2.5">
        <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-accent text-[12px] font-bold text-white">
          {{ (user.display_name || user.email || 'A')[0]?.toUpperCase() }}
        </div>
        <div class="min-w-0">
          <div class="truncate text-[12px] font-semibold">{{ user.display_name || user.email || 'Admin' }}</div>
          <div class="truncate text-[10px] text-mute">{{ user.role || '' }}</div>
        </div>
        <button class="ml-auto text-mute hover:text-bad" title="Logout" @click="logout">
          <svg width="17" height="17" viewBox="0 0 24 24" class="rotate-180"><path d="M9 4H5v16h4M15 8l4 4-4 4M19 12H9" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>
        </button>
      </div>
    </aside>

    <!-- mobile: mini header + page -->
    <div class="md:pl-60">
      <header class="frost sticky top-0 z-20 flex items-center gap-3 px-4 py-2.5 md:hidden">
        <div class="flex h-8 w-8 items-center justify-center rounded-[10px] bg-accent">
          <svg width="16" height="16" viewBox="0 0 24 24" class="text-white"><path fill="none" stroke="currentColor" stroke-width="2" d="M4 6h16v4H4zM4 12h16v4H4zM4 18h10"/></svg>
        </div>
        <div class="min-w-0">
          <div class="truncate text-[14px] font-bold leading-tight">{{ page.label }}</div>
          <div class="text-[10px] text-mute">Gateway Console</div>
        </div>
        <button class="ml-auto flex h-8 w-8 items-center justify-center rounded-full bg-panel/80" title="Logout" @click="logout">
          <svg width="16" height="16" viewBox="0 0 24 24" class="rotate-180 text-ink"><path d="M9 4H5v16h4M15 8l4 4-4 4M19 12H9" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"/></svg>
        </button>
      </header>

      <main class="mx-auto w-full max-w-3xl px-4 pb-6 pt-4 md:max-w-none md:px-8 md:py-7">
        <RouterView />
      </main>

      <!-- iOS-style bottom tab bar (mobile) -->
      <nav class="fixed bottom-0 left-0 right-0 z-20 border-t md:hidden" style="border-color: var(--color-line); background: rgba(255,255,255,0.9); backdrop-filter: saturate(180%) blur(18px); padding-bottom: env(safe-area-inset-bottom)">
        <div class="grid grid-cols-5">
          <RouterLink v-for="n in nav" :key="n.name" :to="n.to"
            class="tappable flex flex-col items-center gap-0.5 py-1.5"
            :class="route.path.startsWith(n.to) ? 'text-accent' : 'text-mute'">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
              <path :d="n.icon" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" stroke-linecap="round" />
            </svg>
            <span class="text-[9px] font-semibold">{{ n.label }}</span>
          </RouterLink>
        </div>
      </nav>
    </div>

    <!-- toast -->
    <div
      v-if="toast.msg"
      class="fixed bottom-20 left-1/2 z-50 max-w-[92vw] -translate-x-1/2 md:bottom-8 md:right-6 md:left-auto md:translate-x-0"
    >
      <div class="rounded-2xl px-4 py-3 text-[13px] font-medium shadow-xl" :class="toast.kind === 'ok' ? 'bg-ink text-white' : 'bg-bad text-white'">
        {{ toast.msg }}
      </div>
    </div>
  </div>
</template>