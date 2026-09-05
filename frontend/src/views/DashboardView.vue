<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { api, auth } from '../lib/api.js'

const router = useRouter()
const apis = ref([])
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  try {
    apis.value = await api.listApis()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
})

const user = computed(() => auth.user || {})
const active = computed(() => apis.value.filter((a) => a.is_active && !a.deleted_at))
const healthy = computed(() => apis.value.filter((a) => a.status === 'healthy'))

const sections = [
  { to: '/apis', title: 'Registry', desc: 'Register & manage APIs', steps: 0, icon: 'M4 4h16v5H4zM4 11h16v5H4zM4 18h16v2H4z', accent: '#0071e3' },
  { to: '/simulate', title: 'Simulate', desc: 'Preview generated nginx config', steps: 1, icon: 'M7 9h10M7 13h10M5 17h14', accent: '#0a84ff' },
  { to: '/monitoring', title: 'Monitor', desc: 'Live traffic & health stats', steps: 2, icon: 'M5 19V9M10 19V5M15 19v-7M20 19V3', accent: '#34c759' },
  { to: '/docs', title: 'Docs', desc: 'Swagger UI & exposed routes', steps: 3, icon: 'M5 4h14v16H5zM9 8h6M9 12h6', accent: '#ff9f0a' }
]

const statCards = computed(() => [
  { label: 'Registered', value: apis.value.length, cls: '' },
  { label: 'Active', value: active.value.length, cls: 'text-ok' },
  { label: 'Healthy', value: healthy.value.length, cls: 'text-accent-2' },
  { label: 'Auth protected', value: apis.value.filter((a) => a.requires_auth).length, cls: 'text-warn' }
])
</script>

<template>
  <div>
    <header class="mb-5">
      <p class="label-sm m-0">Dashboard</p>
      <h1 class="m-0 text-[22px] font-bold tracking-tight">
        Selamat datang, {{ user.display_name || user.email || 'admin' }}
      </h1>
    </header>

    <p v-if="error" class="text-bad">{{ error }}</p>

    <div class="grid grid-cols-4 gap-2 py-2">
      <div v-if="!loading" v-for="s in statCards" :key="s.label" class="flex flex-col items-center rounded-[16px] bg-panel py-3 shadow-sm">
        <div class="text-[19px] font-bold" :class="s.cls">{{ s.value }}</div>
        <div class="label-sm pt-0.5">{{ s.label }}</div>
      </div>
      <template v-if="loading">
        <div v-for="i in 4" :key="i" class="h-[62px] animate-pulse rounded-[16px] bg-panel/70"></div>
      </template>
    </div>

    <div class="mt-4 grid grid-cols-2 gap-3 md:grid-cols-4">
      <button
        v-for="s in sections"
        :key="s.title"
        class="tappable overflow-hidden rounded-[18px] bg-panel p-0 text-left shadow-sm"
        @click="router.push(s.to)"
      >
        <div class="flex items-center justify-between px-4 pt-4">
          <span class="flex h-9 w-9 items-center justify-center rounded-xl" :style="{ background: s.accent + '1a', color: s.accent }">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
              <path :d="s.icon" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" stroke-linecap="round" />
            </svg>
          </span>
          <svg width="16" height="16" viewBox="0 0 24 24" class="text-mute"><path fill="none" stroke="currentColor" stroke-width="2.2" d="M9 6l6 6-6 6"/></svg>
        </div>
        <div class="px-4 pb-4 pt-3">
          <div class="text-[14px] font-bold">{{ s.title }}</div>
          <div class="mt-0.5 text-[11px] leading-snug text-mute">{{ s.desc }}</div>
        </div>
      </button>
    </div>

    <div class="label-sm mb-2 mt-6">Recently registered</div>
    <div class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
      <template v-if="apis.length">
        <div v-for="a in apis.slice(0, 4)" :key="a.id" class="inset-cell tappable" @click="router.push(`/apis/${a.id}/edit`)">
          <div class="min-w-0">
            <div class="truncate text-[14px] font-semibold">{{ a.name }}</div>
            <div class="truncate text-[11px] text-mute">{{ a.base_path }} → {{ a.upstream.replace(/^https?:\/\//, '') }}</div>
          </div>
          <span class="shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold" :class="a.status === 'healthy' ? 'bg-ok/10 text-ok' : 'bg-panel-2 text-mute'">
            {{ a.status }}
          </span>
        </div>
      </template>
      <div v-else-if="!loading" class="px-4 py-10 text-center text-mute">No APIs registered yet.</div>
    </div>
  </div>
</template>