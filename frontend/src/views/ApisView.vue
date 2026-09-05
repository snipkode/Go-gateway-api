<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { show } from '../toast'

const apis = ref([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    apis.value = await api.listApis()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function del(a) {
  if (!confirm(`Remove ${a.name} (${a.base_path}) from the gateway?`)) return
  try {
    await api.deleteApi(a.id)
    show(`"${a.base_path}" removed · gateway reloaded`)
    load()
  } catch (e) {
    show(e.message, 'err')
  }
}

async function restore(id) {
  try {
    await api.restoreApi(id)
    show('API restored · gateway reloaded')
    load()
  } catch (e) {
    show(e.message, 'err')
  }
}

const statusClass = (s) =>
  ({
    healthy: 'bg-ok/15 text-ok',
    unhealthy: 'bg-bad/15 text-bad',
    unknown: 'bg-panel-2 text-mute'
  })[s] || 'bg-panel-2 text-mute'
</script>

<template>
  <div>
    <div class="flex items-center mb-5">
      <div>
        <h1 class="m-0 text-xl font-semibold">API Registry</h1>
        <p class="text-mute mt-1 m-0">
          Register upstream APIs to expose & protect them through the gateway.
        </p>
      </div>
      <span class="flex-1"></span>
      <RouterLink to="/apis/new" class="rounded-lg bg-accent px-4 py-2 text-sm text-white no-underline hover:brightness-110">
        + Register API
      </RouterLink>
    </div>

    <p v-if="error" class="text-bad">{{ error }}</p>
    <p v-if="loading" class="text-mute">Loading…</p>

    <div v-else-if="apis.length" class="overflow-x-auto rounded-xl border border-line bg-panel">
      <table class="w-full border-collapse">
        <thead>
          <tr class="text-left text-[11px] uppercase tracking-wide text-mute">
            <th class="border-b border-line px-5 py-3">Name / Path</th>
            <th class="border-b border-line px-5 py-3">Upstream</th>
            <th class="border-b border-line px-5 py-3">Methods</th>
            <th class="border-b border-line px-5 py-3">Auth</th>
            <th class="border-b border-line px-5 py-3">RPM</th>
            <th class="border-b border-line px-5 py-3">Status</th>
            <th class="border-b border-line px-5 py-3">Active</th>
            <th class="border-b border-line px-5 py-3 text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in apis" :key="a.id">
            <td class="border-b border-line px-5 py-3">
              <div class="font-medium">{{ a.name }}</div>
              <code class="rounded-md bg-panel-2 px-2 py-0.5 text-xs text-accent-2">{{ a.base_path }}/*</code>
            </td>
            <td class="border-b border-line px-5 py-3 text-mute">{{ a.upstream }}</td>
            <td class="border-b border-line px-5 py-3 text-xs">{{ a.methods.join(', ') }}</td>
            <td class="border-b border-line px-5 py-3">
              <span class="rounded-full px-2.5 py-1 text-[11px] font-semibold" :class="a.requires_auth ? 'bg-accent/15 text-accent-2' : 'bg-panel-2 text-mute'">
                {{ a.requires_auth ? 'JWT' : 'open' }}
              </span>
            </td>
            <td class="border-b border-line px-5 py-3">{{ a.rate_limit_rpm }}</td>
            <td class="border-b border-line px-5 py-3">
              <span class="rounded-full px-2.5 py-1 text-[11px] font-semibold" :class="statusClass(a.status)">
                {{ a.status }}
              </span>
            </td>
            <td class="border-b border-line px-5 py-3">
              <span class="rounded-full px-2.5 py-1 text-[11px] font-semibold" :class="a.is_active ? 'bg-ok/15 text-ok' : 'bg-panel-2 text-mute'">
                {{ a.is_active ? 'ON' : 'OFF' }}
              </span>
            </td>
            <td class="border-b border-line px-5 py-3 text-right">
              <RouterLink :to="`/apis/${a.id}/edit`" class="mr-2 rounded-md border border-line bg-panel-2 px-2.5 py-1 text-xs no-underline hover:brightness-115">
                Edit
              </RouterLink>
              <RouterLink :to="`/simulate?id=${a.id}`" class="mr-2 rounded-md border border-line bg-panel-2 px-2.5 py-1 text-xs no-underline hover:brightness-115">
                Simulate
              </RouterLink>
              <button v-if="!a.deleted_at" class="rounded-md border border-bad px-2.5 py-1 text-xs text-bad hover:brightness-115" @click="del(a)">
                Delete
              </button>
              <button v-else class="rounded-md border border-ok px-2.5 py-1 text-xs text-ok hover:brightness-115" @click="restore(a.id)">
                Restore
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else class="rounded-xl border border-line bg-panel py-14 text-center text-mute">
      <p>Nothing registered yet — the gateway only exposes what's in this table.</p>
      <RouterLink to="/apis/new" class="inline-block mt-2 rounded-lg bg-accent px-4 py-2 text-white no-underline">
        Register your first API
      </RouterLink>
    </div>
  </div>
</template>