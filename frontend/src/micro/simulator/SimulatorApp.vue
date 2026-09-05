<script setup>
import { ref, onMounted, computed } from 'vue'
import { api, auth } from '../../lib/api.js'
import { show } from '../../lib/toast.js'

const props = defineProps({ init: { type: String, default: '' } })

const apis = ref([])
const selectedId = ref(props.init || '')
const snippet = ref('')
const previewing = ref(false)
const publishing = ref(false)
const error = ref('')

// ── request tester state ──
const subpath = ref('healthz')
const method = ref('GET')
const attachJwt = ref(true)
const bodyText = ref('')
const sending = ref(false)
const resp = ref(null)
const tokenText = ref('')
const tokenCopied = ref(false)
const allMethodBody = ['POST', 'PUT', 'PATCH']

onMounted(async () => {
  try {
    apis.value = await api.listApis()
    if (!selectedId.value) selectedId.value = String(apis.value[0]?.id || '')
    if (selectedId.value) { preview(); autoSend() }
  } catch (e) {
    error.value = e.message
  }
})

const selected = computed(() => apis.value.find((a) => a.id === Number(selectedId.value)))

function grabToken() {
  tokenText.value = auth.token
  if (navigator.clipboard?.writeText) {
    navigator.clipboard.writeText(auth.token).then(() => {
      tokenCopied.value = true
      setTimeout(() => (tokenCopied.value = false), 1600)
    }).catch(() => {})
  }
  show(tokenText.value ? 'Session token copied to clipboard' : 'No session — sign in first')
}

async function preview() {
  if (!selectedId.value) return
  previewing.value = true
  try {
    const data = await api.previewApi(selectedId.value)
    snippet.value = data?.config ?? data ?? ''
  } catch (e) {
    show('Preview failed: ' + e.message)
  } finally {
    previewing.value = false
  }
}

async function publish() {
  publishing.value = true
  try {
    await api.publish()
    show('Gateway config published & reloaded')
  } catch (e) {
    show('Publish failed: ' + e.message)
  } finally {
    publishing.value = false
  }
}

function onSelect() {
  preview()
  autoSend()
}

function autoSend() {
  setTimeout(() => send(), 350)
}

function buildUrl() {
  const base = selected.value.base_path.replace(/\/+$/, '')
  const sub = (subpath.value || '').replace(/^\/+/, '')
  return window.location.origin + base + (sub ? '/' + sub : '')
}

async function send() {
  if (!selectedId.value) return
  sending.value = true
  resp.value = null
  const url = buildUrl()
  const headers = { Accept: 'application/json' }
  if (allMethodBody.includes(method.value)) headers['Content-Type'] = 'application/json'
  if (attachJwt.value && auth.token) headers['Authorization'] = 'Bearer ' + auth.token
  const started = performance.now()
  const ctrl = new AbortController()
  const t = setTimeout(() => ctrl.abort(), 10000)
  try {
    const r = await fetch(url, { method: method.value, headers, body: allMethodBody.includes(method.value) && bodyText.value ? bodyText.value : undefined, signal: ctrl.signal })
    const text = await r.text()
    resp.value = { ok: r.ok, status: r.status, ms: Math.round(performance.now() - started), size: text.length, text, url }
  } catch (e) {
    resp.value = { ok: false, status: 0, ms: Math.round(performance.now() - started), size: 0, text: e.name === 'AbortError' ? 'Request timed out after 10s' : `Failed to send: ${e.message}`, url }
  } finally {
    clearTimeout(t)
    sending.value = false
  }
}

const curlDoc = computed(() => {
  if (!selected.value) return ''
  const url = `${window.location.origin}${selected.value.base_path.replace(/\/+$/, '')}/healthz`
  const authFlag = selected.value.methods.includes('POST') ? '-X POST' : ''
  const jwt = selected.value.requires_auth ? ` -H "Authorization: Bearer $TOKEN"` : ''
  return `# 1) get a token (demo admin)
TOKEN=$(curl -s -X POST ${window.location.origin}/api/v1/auth/login \\
  -H "Content-Type: application/json" \\
  -d '{"email":"admin@example.com","password":"admin123"}' | jq -r .data.access_token)

# 2) call the registered API through the gateway
curl -s ${selected.value.methods[0]}${url}${authFlag}${jwt}`
})

function pretty(respText) {
  try { return JSON.stringify(JSON.parse(respText), null, 2) } catch { return respText }
}
</script>

<template>
  <div>
    <header class="mb-4">
      <h1 class="m-0 text-[20px] font-bold tracking-tight">Simulate & Test</h1>
      <p class="label-sm m-0 mt-0.5">Preview config · send real requests · copy the curl</p>
    </header>

    <p v-if="error" class="text-bad">{{ error }}</p>

    <!-- ── API selector + meta ── -->
    <div class="mb-4 rounded-[18px] bg-panel p-4 shadow-sm">
      <label class="label-sm block pb-2">Registered API</label>
      <select v-model="selectedId" @change="onSelect"
        class="w-full rounded-xl border bg-panel-2 px-3 py-2.5 text-[14px] focus:outline-none"
        :style="{ borderColor: 'var(--color-line)' }">
        <option v-for="a in apis" :key="a.id" :value="a.id">{{ a.name }} — {{ a.base_path }}</option>
      </select>

      <!-- 2-col on sm+, 1-col on xs -->
      <div class="mt-3 grid grid-cols-1 gap-2 text-[12px] min-[400px]:grid-cols-2">
        <div class="rounded-xl bg-panel-2 px-3 py-2">
          <div class="label-sm">Upstream</div>
          <div class="break-all pt-0.5">{{ selected?.upstream }}</div>
        </div>
        <div class="rounded-xl bg-panel-2 px-3 py-2">
          <div class="label-sm">Protection</div>
          <div class="pt-0.5">{{ selected?.requires_auth ? 'JWT · auth_request' : 'open' }}</div>
        </div>
        <div class="rounded-xl bg-panel-2 px-3 py-2">
          <div class="label-sm">Methods</div>
          <div class="break-all pt-0.5">{{ selected?.methods.join(', ') }}</div>
        </div>
        <div class="rounded-xl bg-panel-2 px-3 py-2">
          <div class="label-sm">Rate / health</div>
          <div class="pt-0.5">{{ selected?.rate_limit_rpm }}r/m · {{ selected?.status }}</div>
        </div>
      </div>

      <!-- stacked on mobile, side-by-side on sm+ -->
      <div class="mt-3 flex flex-col gap-2 min-[400px]:flex-row">
        <button class="btn-ghost tappable flex-1" :disabled="previewing" @click="preview">
          {{ previewing ? 'Rendering…' : '↻ Preview config' }}
        </button>
        <button class="btn-primary tappable flex-1" :disabled="publishing" @click="publish">
          {{ publishing ? 'Publishing…' : 'Apply & reload' }}
        </button>
      </div>
    </div>

    <!-- ── send a test request ── -->
    <div class="mb-4 rounded-[18px] bg-panel p-4 shadow-sm">
      <div class="label-sm pb-2">
        Send a test request
        <span class="text-mute">
          (through the gateway → upstream {{ selected?.requires_auth ? '· with jwt' : '· open' }})
        </span>
      </div>

      <!-- base_path badge stacked above the subpath input on very narrow screens -->
      <div class="flex flex-col gap-1.5 min-[400px]:flex-row min-[400px]:items-center min-[400px]:gap-2">
        <code class="inline-block shrink-0 rounded-lg bg-panel-2 px-2 py-1.5 text-[12px] text-accent leading-snug break-all">{{ selected?.base_path }}</code>
        <input v-model="subpath" type="text" placeholder="healthz"
          class="w-full rounded-xl border bg-panel-2 px-3 py-2 text-[13px] focus:outline-none"
          :style="{ borderColor: 'var(--color-line)' }" />
      </div>

      <!-- method seg: allow wrapping so many methods don't overflow -->
      <div class="seg mt-2.5 flex-wrap">
        <button v-for="m in selected?.methods || ['GET']" :key="m" type="button"
          :class="{ on: method === m }" @click="method = m">{{ m }}</button>
      </div>

      <textarea v-if="allMethodBody.includes(method)" v-model="bodyText" rows="3"
        placeholder='JSON body, e.g. {"name":"x"}'
        class="mt-2 w-full rounded-xl border bg-panel-2 px-3 py-2 text-[12px] focus:outline-none"
        :style="{ borderColor: 'var(--color-line)' }"></textarea>

      <!-- JWT toggle row + action buttons: wrap naturally, full-width on xs -->
      <div class="mt-2.5 flex flex-wrap items-center gap-2">
        <label class="flex items-center gap-2 text-[12px]">
          <input v-model="attachJwt" type="checkbox" class="accent-[#0071e3]" />
          Attach session JWT
        </label>
        <button class="btn-ghost tappable !py-2 text-[12px] ml-auto" @click="grabToken">
          {{ tokenCopied ? 'Copied ✓' : tokenText ? 'Refresh token' : 'Get token' }}
        </button>
        <button class="btn-primary tappable w-full min-[400px]:w-auto"
          :disabled="sending || !selectedId" @click="send">
          {{ sending ? 'Sending…' : '▶ Send' }}
        </button>
      </div>

      <!-- JWT preview strip -->
      <div v-if="tokenText" class="mt-2 flex items-center gap-2">
        <code class="thin-scroll block min-w-0 flex-1 overflow-x-auto whitespace-nowrap rounded-lg bg-panel-2 px-2.5 py-1.5 text-[11px] text-accent">{{ tokenText.slice(0, 28) }}…{{ tokenText.slice(-12) }}</code>
        <span class="shrink-0 text-[10px] text-ok">JWT ready</span>
      </div>

      <!-- response panel -->
      <div v-if="resp" class="mt-3 overflow-hidden rounded-xl border" :style="{ borderColor: 'var(--color-line)' }">
        <!-- status bar: wrap on narrow screens -->
        <div class="flex flex-wrap items-center gap-x-2 gap-y-1 border-b px-3 py-2 text-[11px]"
          :style="{ borderColor: 'var(--color-line)' }">
          <span class="rounded-full px-2 py-0.5 font-bold shrink-0"
            :class="resp.status === 0 ? 'bg-panel-2 text-mute' : resp.status < 400 ? 'bg-ok/10 text-ok' : 'bg-bad/10 text-bad'">
            {{ resp.status || 'ERR' }}
          </span>
          <span class="shrink-0 text-mute">{{ resp.ms }}ms · {{ resp.size }}B</span>
          <span class="w-full truncate text-mute">{{ resp.url }}</span>
        </div>
        <pre class="thin-scroll m-0 max-h-[280px] overflow-auto px-3 py-2.5 text-[11px] leading-relaxed">{{ pretty(resp.text) || '(empty body)' }}</pre>
      </div>
    </div>

    <!-- ── curl documentation ── -->
    <div class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
      <div class="flex items-center gap-2 px-4 py-3">
        <span class="h-2 w-2 rounded-full bg-ok shrink-0"></span>
        <span class="text-[11px] text-mute">How to call this API from outside the console (curl)</span>
      </div>
      <pre v-if="curlDoc" class="thin-scroll m-0 max-h-[260px] overflow-x-auto border-t bg-panel-2 px-4 py-3 text-[11px] leading-relaxed"
        :style="{ borderColor: 'var(--color-line)' }">{{ curlDoc }}</pre>
      <div v-else class="border-t px-4 py-10 text-center text-mute"
        :style="{ borderColor: 'var(--color-line)' }">
        Select an API to see its curl example.
      </div>
    </div>

    <!-- ── generated config ── -->
    <div class="mt-4 overflow-hidden rounded-[18px] bg-panel shadow-sm">
      <div class="flex items-center gap-2 px-4 py-3">
        <span class="h-2 w-2 rounded-full bg-ok shrink-0"></span>
        <span class="min-w-0 truncate text-[11px] text-mute">Generated nginx · locations/reg_{{ selectedId }}.conf</span>
      </div>
      <pre v-if="snippet" class="thin-scroll m-0 max-h-[360px] overflow-x-auto border-t px-4 py-3 text-[11px] leading-relaxed"
        :style="{ borderColor: 'var(--color-line)' }">{{ snippet }}</pre>
      <div v-else class="border-t px-4 py-10 text-center text-mute"
        :style="{ borderColor: 'var(--color-line)' }">
        Select an API to preview its generated config.
      </div>
    </div>
  </div>
</template>