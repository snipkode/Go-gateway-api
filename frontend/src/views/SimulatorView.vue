<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import { show } from '../toast'

const route = useRoute()
const apis = ref([])
const selectedId = ref('')
const snippet = ref('')
const loading = ref(false)
const previewing = ref(false)
const error = ref('')

onMounted(async () => {
  try {
    apis.value = await api.listApis()
    selectedId.value = String(route.query.id || apis.value[0]?.id || '')
    if (selectedId.value) await preview()
  } catch (e) {
    error.value = e.message
  }
})

const selected = computed(() => apis.value.find((a) => a.id === Number(selectedId.value)))

async function preview() {
  if (!selectedId.value) return
  previewing.value = true
  snippet.value = ''
  try {
    snippet.value = await api.previewApi(selectedId.value)
  } catch (e) {
    show(e.message, 'err')
  } finally {
    previewing.value = false
  }
}

async function publish() {
  loading.value = true
  try {
    await api.publish()
    show('Gateway config re-published · nginx reloaded')
    await preview()
  } catch (e) {
    show(e.message, 'err')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <h1 class="m-0 text-xl font-semibold">Configure & Simulate</h1>
    <p class="text-mute mt-1 mb-5">
      Preview the exact nginx config the gateway will apply — then publish it.
    </p>

    <div class="mb-5 flex flex-wrap items-center gap-3">
      <label class="text-xs text-mute">Registered API</label>
      <select v-model="selectedId" @change="preview"
        class="rounded-lg border border-line bg-panel-2 px-3 py-2 text-sm focus:border-accent focus:outline-none">
        <option v-for="a in apis" :key="a.id" :value="a.id">
          {{ a.name }} — {{ a.base_path }}
        </option>
      </select>
      <button class="rounded-lg border border-line bg-panel-2 px-3.5 py-2 text-sm hover:brightness-115" @click="preview" :disabled="previewing">
        {{ previewing ? 'Rendering…' : 'Preview config' }}
      </button>
      <span class="flex-1"></span>
      <button class="rounded-lg bg-accent px-4 py-2 text-sm text-white hover:brightness-110 disabled:opacity-50" :disabled="loading" @click="publish">
        {{ loading ? 'Publishing…' : 'Apply & reload gateway' }}
      </button>
    </div>

    <p v-if="error" class="text-bad">{{ error }}</p>

    <div class="grid grid-cols-3 gap-4">
      <div v-if="selected" class="rounded-xl border border-line bg-panel p-5">
        <h3 class="m-0 text-sm font-semibold text-mute mb-3">Effective configuration</h3>
        <dl class="m-0 space-y-2 text-sm">
          <div class="grid grid-cols-[120px_1fr] gap-2">
            <dt class="text-mute">Path</dt>
            <dd><code class="rounded bg-panel-2 px-1.5 py-0.5 text-xs text-accent-2">{{ selected.base_path }}/*</code></dd>
          </div>
          <div class="grid grid-cols-[120px_1fr] gap-2">
            <dt class="text-mute">Upstream</dt>
            <dd class="break-all">{{ selected.upstream }}</dd>
          </div>
          <div class="grid grid-cols-[120px_1fr] gap-2">
            <dt class="text-mute">Methods</dt>
            <dd>{{ selected.methods.join(', ') }}</dd>
          </div>
          <div class="grid grid-cols-[120px_1fr] gap-2">
            <dt class="text-mute">Protection</dt>
            <dd>
              <span class="rounded-full px-2 py-0.5 text-[11px] font-semibold" :class="selected.requires_auth ? 'bg-accent/15 text-accent-2' : 'bg-panel-2 text-mute'">
                {{ selected.requires_auth ? 'JWT auth_request' : 'open' }}
              </span>
            </dd>
          </div>
          <div class="grid grid-cols-[120px_1fr] gap-2">
            <dt class="text-mute">Rate limit</dt>
            <dd>{{ selected.rate_limit_rpm }} req/min per IP</dd>
          </div>
          <div class="grid grid-cols-[120px_1fr] gap-2">
            <dt class="text-mute">Health</dt>
            <dd>{{ selected.status }}</dd>
          </div>
        </dl>
      </div>

      <div class="col-span-2">
        <div class="rounded-xl border border-line bg-panel">
          <div class="flex items-center gap-2 border-b border-line px-5 py-3">
            <span class="h-2.5 w-2.5 rounded-full bg-ok/70"></span>
            <span class="text-xs text-mute">Generated nginx location block</span>
            <span class="flex-1"></span>
            <span class="text-xs text-mute">locations/reg_{{ selectedId }}.conf</span>
          </div>
          <pre class="mono m-0 overflow-x-auto p-5 text-xs leading-relaxed" v-if="snippet">{{ snippet }}</pre>
          <div v-else class="p-10 text-center text-mute">Select an API to preview its generated config.</div>
        </div>
      </div>
    </div>
  </div>
</template>