<script setup>
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { api } from '../api'
import { show } from '../toast'

const apis = ref([])
const selectedId = ref('')
const stats = ref(null)
const error = ref('')
let timer = null

onMounted(async () => {
  try {
    const list = await api.listApis()
    apis.value = list
    selectedId.value = String(list[0]?.id || '')
    if (selectedId.value) await loadStats()
    if (selectedId.value) timer = setInterval(loadStats, 5000)
  } catch (e) {
    error.value = e.message
  }
})
onUnmounted(() => clearInterval(timer))

watch(selectedId, async (id) => {
  clearInterval(timer)
  if (id) {
    await loadStats()
    timer = setInterval(loadStats, 5000)
  }
})

async function loadStats() {
  if (!selectedId.value) return
  try {
    stats.value = await api.statsApi(selectedId.value)
  } catch (e) {
    show(e.message, 'err')
  }
}

const selected = computed(() => apis.value.find((a) => a.id === Number(selectedId.value)))
const statusClass = (s) =>
  s >= 500 ? 'bg-bad/15 text-bad' : s >= 400 ? 'bg-warn/15 text-warn' : s >= 200 ? 'bg-ok/15 text-ok' : 'bg-panel-2 text-mute'
</script>

<template>
  <div>
    <h1 class="m-0 text-xl font-semibold">Monitoring</h1>
    <p class="text-mute mt-1 mb-5">
      Live traffic for each registered API (nginx access-log → Redis counters, refresh 5s).
    </p>

    <div class="mb-5 flex items-center gap-3">
      <label class="text-xs text-mute">Registered API</label>
      <select v-model="selectedId"
        class="rounded-lg border border-line bg-panel-2 px-3 py-2 text-sm focus:border-accent focus:outline-none">
        <option v-for="a in apis" :key="a.id" :value="a.id">{{ a.name }} — {{ a.base_path }}</option>
      </select>
      <span class="ml-auto inline-flex items-center gap-1.5 text-xs text-mute">
        <span class="h-2 w-2 rounded-full bg-ok animate-pulse"></span> live
      </span>
    </div>

    <p v-if="error" class="text-bad">{{ error }}</p>

    <template v-if="stats">
      <div class="mb-5 grid grid-cols-2 gap-4 md:grid-cols-5">
        <div class="rounded-xl border border-line bg-panel p-4">
          <div class="text-2xl font-bold">{{ stats.today.count }}</div>
          <div class="text-mute text-xs mt-0.5">Requests today</div>
        </div>
        <div class="rounded-xl border border-line bg-panel p-4">
          <div class="text-2xl font-bold text-ok">{{ stats.today['2xx'] }}</div>
          <div class="text-mute text-xs mt-0.5">2xx</div>
        </div>
        <div class="rounded-xl border border-line bg-panel p-4">
          <div class="text-2xl font-bold text-warn">{{ stats.today['4xx'] }}</div>
          <div class="text-mute text-xs mt-0.5">4xx</div>
        </div>
        <div class="rounded-xl border border-line bg-panel p-4">
          <div class="text-2xl font-bold text-bad">{{ stats.today['5xx'] }}</div>
          <div class="text-mute text-xs mt-0.5">5xx</div>
        </div>
        <div class="rounded-xl border border-line bg-panel p-4">
          <div class="text-2xl font-bold">{{ stats.today.latency_ms }}ms</div>
          <div class="text-mute text-xs mt-0.5">Avg latency</div>
        </div>
      </div>

      <div class="rounded-xl border border-line bg-panel">
        <div class="border-b border-line px-5 py-3 text-xs text-mute">Recent requests (last {{ stats.recent.length }})</div>
        <div class="overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="text-left text-[11px] uppercase tracking-wide text-mute">
                <th class="border-b border-line px-5 py-2.5">Time</th>
                <th class="border-b border-line px-5 py-2.5">Status</th>
                <th class="border-b border-line px-5 py-2.5">Method</th>
                <th class="border-b border-line px-5 py-2.5">URI</th>
                <th class="border-b border-line px-5 py-2.5">Latency</th>
                <th class="border-b border-line px-5 py-2.5">Client</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(r, i) in stats.recent.slice().reverse()" :key="i">
                <td class="border-b border-line px-5 py-2.5 text-mute text-xs">{{ r.time?.slice(11) }}</td>
                <td class="border-b border-line px-5 py-2.5">
                  <span class="rounded-full px-2 py-0.5 text-[11px] font-semibold" :class="statusClass(r.status)">{{ r.status }}</span>
                </td>
                <td class="border-b border-line px-5 py-2.5 text-xs">{{ r.method }}</td>
                <td class="border-b border-line px-5 py-2.5 text-xs break-all">{{ r.uri }}</td>
                <td class="border-b border-line px-5 py-2.5 text-xs">{{ (r.rt * 1000).toFixed(0) }}ms</td>
                <td class="border-b border-line px-5 py-2.5 text-mute text-xs">{{ r.ip }}</td>
              </tr>
              <tr v-if="!stats.recent.length">
                <td colspan="6" class="px-5 py-8 text-center text-mute">No proxied requests yet — hit the API to see traffic.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
    <div v-else class="muted">Select an API…</div>
  </div>
</template>