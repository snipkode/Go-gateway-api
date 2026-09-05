<script setup>
import { ref, onMounted, computed, onBeforeUnmount } from 'vue'
import { api, auth } from '../../lib/api.js'
import { show } from '../../lib/toast.js'

const props = defineProps({ init: { type: String, default: '' } })

const apis = ref([])
const selectedId = ref(props.init || '')
const snippet = ref('')
const previewing = ref(false)
const publishing = ref(false)
const error = ref('')

// ── custom dropdown ──
const dropOpen = ref(false)
const dropSearch = ref('')
const dropRef = ref(null)

const filteredApis = computed(() => {
  const q = dropSearch.value.trim().toLowerCase()
  if (!q) return apis.value
  return apis.value.filter(
    (a) => a.name.toLowerCase().includes(q) || a.base_path.toLowerCase().includes(q)
  )
})

function pickApi(id) {
  selectedId.value = String(id)
  dropOpen.value = false
  dropSearch.value = ''
  onSelect()
}

function onClickOutside(e) {
  if (dropRef.value && !dropRef.value.contains(e.target)) {
    dropOpen.value = false
    dropSearch.value = ''
  }
}

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
  document.addEventListener('mousedown', onClickOutside)
  try {
    apis.value = await api.listApis()
    if (!selectedId.value) selectedId.value = String(apis.value[0]?.id || '')
    if (selectedId.value) { preview(); autoSend() }
  } catch (e) {
    error.value = e.message
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onClickOutside)
})

const selected = computed(() => apis.value.find((a) => a.id === Number(selectedId.value)))

// badge colour per method
function methodColor(m) {
  return {
    GET: 'text-[#34c759] bg-[#34c759]/10',
    POST: 'text-[#0071e3] bg-[#0071e3]/10',
    PUT: 'text-[#ff9f0a] bg-[#ff9f0a]/10',
    PATCH: 'text-[#ff9f0a] bg-[#ff9f0a]/10',
    DELETE: 'text-[#ff3b30] bg-[#ff3b30]/10',
  }[m] ?? 'text-mute bg-panel-2'
}

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
      <div class="label-sm pb-2">Registered API</div>

      <!-- Custom dropdown trigger -->
      <div ref="dropRef" class="relative">
        <button
          type="button"
          class="tappable flex w-full items-center gap-3 rounded-xl border bg-panel-2 px-3 py-2.5 text-left transition-colors hover:bg-[var(--color-accent-tint)]"
          :style="{ borderColor: dropOpen ? 'var(--color-accent)' : 'var(--color-line)' }"
          @click="dropOpen = !dropOpen"
        >
          <!-- selected state -->
          <div v-if="selected" class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="truncate text-[14px] font-medium">{{ selected.name }}</span>
              <span v-if="selected.requires_auth" class="shrink-0 rounded-full bg-[#0071e3]/10 px-1.5 py-0.5 text-[10px] font-semibold text-[#0071e3]">JWT</span>
              <span v-else class="shrink-0 rounded-full bg-ok/10 px-1.5 py-0.5 text-[10px] font-semibold text-ok">open</span>
            </div>
            <div class="truncate text-[11px] text-mute">{{ selected.base_path }}</div>
          </div>
          <div v-else class="flex-1 text-[14px] text-mute">Pilih API…</div>

          <!-- chevron -->
          <svg class="shrink-0 text-mute transition-transform duration-200" :class="{ 'rotate-180': dropOpen }" width="14" height="14" viewBox="0 0 14 14" fill="none">
            <path d="M3 5l4 4 4-4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </button>

        <!-- Dropdown popup -->
        <Transition
          enter-active-class="transition duration-150 ease-out"
          enter-from-class="opacity-0 translate-y-1 scale-[0.98]"
          enter-to-class="opacity-100 translate-y-0 scale-100"
          leave-active-class="transition duration-100 ease-in"
          leave-from-class="opacity-100 translate-y-0 scale-100"
          leave-to-class="opacity-0 translate-y-1 scale-[0.98]"
        >
          <div
            v-if="dropOpen"
            class="absolute left-0 right-0 top-[calc(100%+6px)] z-50 overflow-hidden rounded-[18px] border bg-panel shadow-xl"
            :style="{ borderColor: 'var(--color-line)', boxShadow: '0 8px 32px rgba(0,0,0,0.12), 0 2px 8px rgba(0,0,0,0.06)' }"
          >
            <!-- search box -->
            <div class="flex items-center gap-2 border-b px-3 py-2.5" :style="{ borderColor: 'var(--color-line)' }">
              <svg class="shrink-0 text-mute" width="14" height="14" viewBox="0 0 14 14" fill="none">
                <circle cx="6" cy="6" r="4.5" stroke="currentColor" stroke-width="1.5"/>
                <path d="M10 10l2.5 2.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              </svg>
              <input
                v-model="dropSearch"
                type="text"
                placeholder="Cari nama atau path…"
                autofocus
                class="flex-1 bg-transparent text-[13px] focus:outline-none"
              />
              <button v-if="dropSearch" class="text-mute hover:text-ink" @click="dropSearch = ''">
                <svg width="13" height="13" viewBox="0 0 13 13" fill="none">
                  <path d="M3 3l7 7M10 3l-7 7" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                </svg>
              </button>
            </div>

            <!-- list -->
            <ul class="thin-scroll max-h-[260px] overflow-y-auto py-1">
              <li v-if="filteredApis.length === 0" class="px-4 py-6 text-center text-[12px] text-mute">
                Tidak ada hasil
              </li>
              <li
                v-for="a in filteredApis"
                :key="a.id"
                class="tappable group flex cursor-pointer items-start gap-3 px-3 py-2.5 transition-colors hover:bg-panel-2"
                :class="{ 'bg-[var(--color-accent-tint)]': String(a.id) === String(selectedId) }"
                @click="pickApi(a.id)"
              >
                <!-- check mark -->
                <span class="mt-0.5 w-4 shrink-0 text-[var(--color-accent)]">
                  <svg v-if="String(a.id) === String(selectedId)" width="14" height="14" viewBox="0 0 14 14" fill="none">
                    <path d="M2.5 7l3.5 3.5 5.5-6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                </span>

                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                    <span class="text-[13px] font-medium">{{ a.name }}</span>
                    <!-- method badges -->
                    <span
                      v-for="m in (a.methods || []).slice(0, 3)"
                      :key="m"
                      class="rounded-full px-1.5 py-0.5 text-[10px] font-bold"
                      :class="methodColor(m)"
                    >{{ m }}</span>
                  </div>
                  <div class="flex flex-wrap items-center gap-2 pt-0.5 text-[11px] text-mute">
                    <code class="text-accent">{{ a.base_path }}</code>
                    <span v-if="a.requires_auth" class="text-[#0071e3]">· JWT</span>
                    <span v-else class="text-ok">· open</span>
                    <span class="ml-auto shrink-0">{{ a.rate_limit_rpm }}r/m</span>
                  </div>
                </div>
              </li>
            </ul>

            <!-- footer count -->
            <div class="border-t px-4 py-2 text-[10px] text-mute" :style="{ borderColor: 'var(--color-line)' }">
              {{ filteredApis.length }} dari {{ apis.length }} API
            </div>
          </div>
        </Transition>
      </div>

      <!-- meta cards -->
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

      <div class="flex flex-col gap-1.5 min-[400px]:flex-row min-[400px]:items-center min-[400px]:gap-2">
        <code class="inline-block shrink-0 rounded-lg bg-panel-2 px-2 py-1.5 text-[12px] text-accent leading-snug break-all">{{ selected?.base_path }}</code>
        <input v-model="subpath" type="text" placeholder="healthz"
          class="w-full rounded-xl border bg-panel-2 px-3 py-2 text-[13px] focus:outline-none"
          :style="{ borderColor: 'var(--color-line)' }" />
      </div>

      <div class="seg mt-2.5 flex-wrap">
        <button v-for="m in selected?.methods || ['GET']" :key="m" type="button"
          :class="{ on: method === m }" @click="method = m">{{ m }}</button>
      </div>

      <textarea v-if="allMethodBody.includes(method)" v-model="bodyText" rows="3"
        placeholder='JSON body, e.g. {"name":"x"}'
        class="mt-2 w-full rounded-xl border bg-panel-2 px-3 py-2 text-[12px] focus:outline-none"
        :style="{ borderColor: 'var(--color-line)' }"></textarea>

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

      <div v-if="tokenText" class="mt-2 flex items-center gap-2">
        <code class="thin-scroll block min-w-0 flex-1 overflow-x-auto whitespace-nowrap rounded-lg bg-panel-2 px-2.5 py-1.5 text-[11px] text-accent">{{ tokenText.slice(0, 28) }}…{{ tokenText.slice(-12) }}</code>
        <!-- clipboard icon button -->
        <button
          class="tappable shrink-0 rounded-lg p-1.5 transition-colors"
          :class="tokenCopied ? 'bg-ok/10 text-ok' : 'bg-panel-2 text-mute hover:text-ink'"
          :title="tokenCopied ? 'Tersalin!' : 'Salin ke clipboard'"
          @click="grabToken"
        >
          <!-- copy icon -->
          <svg v-if="!tokenCopied" width="14" height="14" viewBox="0 0 14 14" fill="none">
            <rect x="4.5" y="1.5" width="8" height="9" rx="1.5" stroke="currentColor" stroke-width="1.4"/>
            <path d="M1.5 4.5H3a1 1 0 011 1v7a1 1 0 001 1h5.5" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
          </svg>
          <!-- check icon -->
          <svg v-else width="14" height="14" viewBox="0 0 14 14" fill="none">
            <path d="M2.5 7l3.5 3.5 5.5-6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </button>
      </div>

      <div v-if="resp" class="mt-3 overflow-hidden rounded-xl border" :style="{ borderColor: 'var(--color-line)' }">
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
