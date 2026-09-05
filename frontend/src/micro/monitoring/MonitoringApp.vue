<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { api } from '../../lib/api.js'

const apis = ref([])
const selectedId = ref('')
const stats = ref(null)
let timer = null

onMounted(async () => {
  try {
    apis.value = await api.listApis()
    selectedId.value = String(apis.value[0]?.id || '')
    if (selectedId.value) await load()
    if (selectedId.value) timer = setInterval(load, 5000)
  } catch {
    /* silent */
  }
})
onUnmounted(() => clearInterval(timer))

watch(selectedId, async (id) => {
  clearInterval(timer)
  if (id) {
    await load()
    timer = setInterval(load, 5000)
  }
})

async function load() {
  if (!selectedId.value) return
  try {
    stats.value = await api.statsApi(selectedId.value)
  } catch {
    /* keep last */
  }
}

const selected = computed(() => apis.value.find((a) => a.id === Number(selectedId.value)))
const statusClass = (s) =>
  s >= 500 ? 'bg-bad/10 text-bad' : s >= 400 ? 'bg-warn/10 text-warn' : s >= 200 ? 'bg-ok/10 text-ok' : 'bg-panel-2 text-mute'

const cards = computed(() => [
  { label: 'Requests', value: String(stats.value?.today.count ?? 0), cls: '' },
  { label: '2xx', value: String(stats.value?.today['2xx'] ?? 0), cls: 'text-ok' },
  { label: '4xx', value: String(stats.value?.today['4xx'] ?? 0), cls: 'text-warn' },
  { label: '5xx', value: String(stats.value?.today['5xx'] ?? 0), cls: 'text-bad' },
  { label: 'Avg ms', value: String(stats.value?.today.latency_ms ?? 0), cls: '' }
])
</script>

<template>
  <div>
    <header class="mb-4 flex items-baseline justify-between">
      <div>
        <h1 class="m-0 text-[20px] font-bold tracking-tight">Monitoring</h1>
        <p class="label-sm m-0 mt-0.5">Live traffic · refreshes every 5s</p>
      </div>
      <span class="inline-flex items-center gap-1.5 text-[11px] text-mute">
        <span ref="dot" class="h-2 w-2 rounded-full bg-ok animate-pulse"></span> live
      </span>
    </header>

    <select v-model="selectedId"
      class="mb-4 w-full rounded-xl border bg-panel px-3 py-2.5 text-[14px] shadow-sm focus:outline-none" :style="{ borderColor: 'var(--color-line)' }">
      <option v-for="a in apis" :key="a.id" :value="a.id">{{ a.name }} — {{ a.base_path }}</option>
    </select>

    <template v-if="stats">
      <div class="grid grid-cols-5 gap-2 py-2">
        <div v-for="c in cards" :key="c.label" class="flex flex-col items-center rounded-[16px] bg-panel py-3 shadow-sm">
          <div class="text-[17px] font-bold" :class="c.cls">{{ c.value }}</div>
          <div class="label-sm pt-0.5">{{ c.label }}</div>
        </div>
      </div>

      <div class="mt-3 overflow-hidden rounded-[18px] bg-panel shadow-sm">
        <div class="label-sm border-b px-4 py-2.5" :style="{ borderColor: 'var(--color-line)' }">
          Recent requests ({{ stats.recent.length }})
        </div>
        <div class="thin-scroll max-h-[52vh] divide-y divide-solid overflow-y-auto" :style="{ borderColor: 'var(--color-line)' }">
          <div v-for="(r, i) in stats.recent.slice().reverse()" :key="i" class="flex items-center gap-2.5 px-4 py-2">
            <span class="w-7 shrink-0 text-center text-[11px] font-bold" :class="statusClass(r.status)">{{ r.status }}</span>
            <div class="min-w-0 flex-1">
              <div class="truncate text-[13px]">{{ r.method }} {{ r.uri }}</div>
              <div class="text-[11px] text-mute">{{ r.time?.slice(11) }} · {{ r.ip }}</div>
            </div>
            <span class="text-[11px] text-mute">{{ (r.rt * 1000).toFixed(0) }}ms</span>
          </div>
          <div v-if="!stats.recent.length" class="px-4 py-10 text-center text-mute">
            No proxied requests for {{ selected?.name }} yet.
          </div>
        </div>
      </div>
    </template>
    <div v-else class="mt-8 text-center text-mute">Select an API to see live traffic.</div>
  </div>
</template>