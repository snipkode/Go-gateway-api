<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'

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

const total = () => apis.value.length
const healthy = () => apis.value.filter((a) => a.status === 'healthy').length
const protectedCount = () => apis.value.filter((a) => a.requires_auth).length
const totalRpm = () => apis.value.reduce((s, a) => s + a.rate_limit_rpm, 0)

const statusBadge = (s) =>
  s === 'healthy' ? 'green' : s === 'unhealthy' ? 'red' : 'gray'
const statusClass = (s) =>
  ({
    green: 'bg-ok/15 text-ok',
    red: 'bg-bad/15 text-bad',
    gray: 'bg-panel-2 text-mute'
  })[statusBadge(s)]
</script>

<template>
  <div>
    <h1 class="m-0 text-xl font-semibold">Dashboard</h1>
    <p class="text-mute mb-5 mt-1">
      Managed upstream APIs exposed through the Nginx gateway.
    </p>

    <p v-if="error" class="text-bad">{{ error }}</p>
    <p v-if="loading" class="text-mute">Loading…</p>

    <template v-else-if="apis.length">
      <div class="mb-5 grid grid-cols-2 gap-4 md:grid-cols-4">
        <div class="rounded-xl border border-line bg-panel p-5">
          <div class="text-3xl font-bold">{{ total() }}</div>
          <div class="text-mute mt-0.5">Registered APIs</div>
        </div>
        <div class="rounded-xl border border-line bg-panel p-5">
          <div class="text-3xl font-bold text-ok">{{ healthy() }}</div>
          <div class="text-mute mt-0.5">Healthy upstreams</div>
        </div>
        <div class="rounded-xl border border-line bg-panel p-5">
          <div class="text-3xl font-bold">{{ protectedCount() }}</div>
          <div class="text-mute mt-0.5">Behind JWT auth</div>
        </div>
        <div class="rounded-xl border border-line bg-panel p-5">
          <div class="text-3xl font-bold">{{ totalRpm() }}</div>
          <div class="text-mute mt-0.5">Rate limit (req/min)</div>
        </div>
      </div>

      <div class="rounded-xl border border-line bg-panel">
        <div class="flex items-center px-5 pt-4">
          <h3 class="m-0 text-sm font-semibold text-mute">Assembly</h3>
          <span class="flex-1"></span>
          <RouterLink to="/apis" class="btn rounded-lg border border-line bg-panel-2 px-3 py-1.5 text-xs no-underline hover:brightness-115">
            Open registry
          </RouterLink>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="text-left text-[11px] uppercase tracking-wide text-mute">
                <th class="border-b border-line px-5 py-3">Path</th>
                <th class="border-b border-line px-5 py-3">Upstream</th>
                <th class="border-b border-line px-5 py-3">Methods</th>
                <th class="border-b border-line px-5 py-3">Status</th>
                <th class="border-b border-line px-5 py-3">Active</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="a in apis.slice(0, 8)" :key="a.id">
                <td class="border-b border-line px-5 py-3">
                  <code class="rounded-md bg-panel-2 px-2 py-0.5 text-xs text-accent-2">{{ a.base_path }}/*</code>
                </td>
                <td class="border-b border-line px-5 py-3 text-mute">{{ a.upstream }}</td>
                <td class="border-b border-line px-5 py-3">{{ a.methods.join(', ') }}</td>
                <td class="border-b border-line px-5 py-3">
                  <span class="rounded-full px-2.5 py-1 text-[11px] font-semibold" :class="statusClass(a.status)">
                    {{ a.status }}
                  </span>
                </td>
                <td class="border-b border-line px-5 py-3">
                  <span
                    class="rounded-full px-2.5 py-1 text-[11px] font-semibold"
                    :class="a.is_active ? 'bg-accent/15 text-accent-2' : 'bg-panel-2 text-mute'"
                  >
                    {{ a.is_active ? 'ON' : 'OFF' }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <div v-else class="py-12 text-center text-mute">
      <p>No registered APIs yet.</p>
      <RouterLink to="/apis/new" class="inline-block rounded-lg bg-accent px-4 py-2 text-white no-underline">
        Register your first API
      </RouterLink>
    </div>
  </div>
</template>