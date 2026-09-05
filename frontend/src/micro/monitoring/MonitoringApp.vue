<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { api } from '../../lib/api.js'

const apis = ref([])
const selectedId = ref('')
const stats = ref(null)
let timer = null

// ── live mode ──
const isLive = ref(true)
const REFRESH_MS = 5000

function startTimer() {
  clearInterval(timer)
  if (isLive.value && selectedId.value) {
    timer = setInterval(load, REFRESH_MS)
  }
}

function toggleLive() {
  isLive.value = !isLive.value
  if (isLive.value) {
    load()
    startTimer()
  } else {
    clearInterval(timer)
  }
}

// ── pagination ──
const PAGE_SIZE = 10
const page = ref(1)

const recentSorted = computed(() => (stats.value?.recent ?? []).slice().reverse())
const totalPages = computed(() => Math.max(1, Math.ceil(recentSorted.value.length / PAGE_SIZE)))
const pageItems = computed(() => {
  const start = (page.value - 1) * PAGE_SIZE
  return recentSorted.value.slice(start, start + PAGE_SIZE)
})

// clamp page on data change
watch(recentSorted, () => {
  if (page.value > totalPages.value) page.value = totalPages.value
})
watch(selectedId, () => { page.value = 1 })

function prevPage() { if (page.value > 1) page.value-- }
function nextPage() { if (page.value < totalPages.value) page.value++ }

// visible page numbers — max 5 slots with ellipsis
const pageNums = computed(() => {
  const t = totalPages.value
  if (t <= 7) return Array.from({ length: t }, (_, i) => i + 1)
  const p = page.value
  const nums = new Set([1, t])
  for (let i = Math.max(2, p - 1); i <= Math.min(t - 1, p + 1); i++) nums.add(i)
  const sorted = [...nums].sort((a, b) => a - b)
  const result = []
  let prev = 0
  for (const n of sorted) {
    if (n - prev > 1) result.push('…')
    result.push(n)
    prev = n
  }
  return result
})

onMounted(async () => {
  try {
    apis.value = await api.listApis()
    selectedId.value = String(apis.value[0]?.id || '')
    if (selectedId.value) { await load(); startTimer() }
  } catch { /* silent */ }
})
onUnmounted(() => clearInterval(timer))

watch(selectedId, async (id) => {
  clearInterval(timer)
  if (id) { await load(); startTimer() }
})

async function load() {
  if (!selectedId.value) return
  try { stats.value = await api.statsApi(selectedId.value) }
  catch { /* keep last */ }
}

const selected = computed(() => apis.value.find((a) => a.id === Number(selectedId.value)))
const statusClass = (s) =>
  s >= 500 ? 'bg-bad/10 text-bad' : s >= 400 ? 'bg-warn/10 text-warn' : s >= 200 ? 'bg-ok/10 text-ok' : 'bg-panel-2 text-mute'

const cards = computed(() => [
  { label: 'Requests', value: String(stats.value?.today.count ?? 0), cls: '' },
  { label: '2xx',      value: String(stats.value?.today['2xx'] ?? 0), cls: 'text-ok' },
  { label: '4xx',      value: String(stats.value?.today['4xx'] ?? 0), cls: 'text-warn' },
  { label: '5xx',      value: String(stats.value?.today['5xx'] ?? 0), cls: 'text-bad' },
  { label: 'Avg ms',   value: String(stats.value?.today.latency_ms ?? 0), cls: '' },
])
</script>

<template>
  <div>
    <header class="mb-4 flex items-start justify-between gap-3">
      <div>
        <h1 class="m-0 text-[20px] font-bold tracking-tight">Monitoring</h1>
        <p class="label-sm m-0 mt-0.5">Live traffic · refreshes every 5s</p>
      </div>

      <!-- live toggle -->
      <button
        class="tappable flex shrink-0 items-center gap-2 rounded-xl border px-3 py-2 text-[12px] font-medium transition-colors"
        :style="{
          borderColor: isLive ? 'var(--color-ok)' : 'var(--color-line)',
          background: isLive ? 'rgba(52,199,89,0.08)' : 'var(--color-panel-2)',
          color: isLive ? 'var(--color-ok)' : 'var(--color-mute)'
        }"
        @click="toggleLive"
      >
        <span class="h-2 w-2 rounded-full transition-colors"
          :class="isLive ? 'bg-ok animate-pulse' : 'bg-[var(--color-line)]'">
        </span>
        {{ isLive ? 'Live' : 'Paused' }}
      </button>
    </header>

    <select v-model="selectedId"
      class="mb-4 w-full rounded-xl border bg-panel px-3 py-2.5 text-[14px] shadow-sm focus:outline-none"
      :style="{ borderColor: 'var(--color-line)' }">
      <option v-for="a in apis" :key="a.id" :value="a.id">{{ a.name }} — {{ a.base_path }}</option>
    </select>

    <template v-if="stats">
      <!-- stat cards -->
      <div class="grid grid-cols-3 gap-1.5 py-2 sm:grid-cols-5 sm:gap-2">
        <div v-for="c in cards" :key="c.label"
          class="flex flex-col items-center rounded-[16px] bg-panel py-3 shadow-sm">
          <div class="text-[17px] font-bold" :class="c.cls">{{ c.value }}</div>
          <div class="label-sm pt-0.5">{{ c.label }}</div>
        </div>
      </div>

      <!-- recent requests -->
      <div class="mt-3 overflow-hidden rounded-[18px] bg-panel shadow-sm">
        <!-- header -->
        <div class="flex items-center justify-between border-b px-4 py-2.5"
          :style="{ borderColor: 'var(--color-line)' }">
          <div class="flex items-center gap-2">
            <span class="label-sm">Recent requests</span>
            <span class="rounded-full bg-panel-2 px-2 py-0.5 text-[11px] text-mute">
              {{ recentSorted.length }}
            </span>
          </div>
          <span class="text-[11px] text-mute">{{ page }} / {{ totalPages }}</span>
        </div>

        <!-- rows -->
        <div class="divide-y" :style="{ '--tw-divide-opacity': 1, borderColor: 'var(--color-line)' }">
          <div v-for="(r, i) in pageItems" :key="i"
            class="flex items-center gap-2.5 px-4 py-2.5 transition-colors hover:bg-panel-2">
            <span class="w-8 shrink-0 text-center text-[11px] font-bold rounded-full py-0.5"
              :class="statusClass(r.status)">{{ r.status }}</span>
            <div class="min-w-0 flex-1">
              <div class="truncate text-[13px]">{{ r.method }} {{ r.uri }}</div>
              <div class="text-[11px] text-mute">{{ r.time?.slice(11) }} · {{ r.ip }}</div>
            </div>
            <span class="shrink-0 text-[11px] text-mute">{{ (r.rt * 1000).toFixed(0) }}ms</span>
          </div>
          <div v-if="!recentSorted.length" class="px-4 py-10 text-center text-[13px] text-mute">
            No proxied requests for <strong>{{ selected?.name }}</strong> yet.
          </div>
        </div>

        <!-- pagination -->
        <div v-if="totalPages > 1"
          class="flex items-center justify-between border-t px-4 py-2.5"
          :style="{ borderColor: 'var(--color-line)' }">
          <button
            class="tappable flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-[12px] hover:bg-panel-2 disabled:opacity-30"
            :disabled="page === 1" @click="prevPage">
            <svg width="13" height="13" viewBox="0 0 13 13" fill="none">
              <path d="M8 2.5L4.5 6.5l3.5 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            Prev
          </button>

          <div class="flex items-center gap-1">
            <template v-for="p in pageNums" :key="p">
              <span v-if="p === '…'" class="w-6 text-center text-[12px] text-mute">…</span>
              <button v-else
                class="tappable h-7 w-7 rounded-lg text-[12px] font-medium transition-colors"
                :class="p === page ? 'bg-[var(--color-accent)] text-white' : 'text-mute hover:bg-panel-2'"
                @click="page = p">{{ p }}</button>
            </template>
          </div>

          <button
            class="tappable flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-[12px] hover:bg-panel-2 disabled:opacity-30"
            :disabled="page === totalPages" @click="nextPage">
            Next
            <svg width="13" height="13" viewBox="0 0 13 13" fill="none">
              <path d="M5 2.5L8.5 6.5 5 10.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
        </div>
      </div>
    </template>

    <div v-else class="mt-8 text-center text-mute">Select an API to see live traffic.</div>
  </div>
</template>
