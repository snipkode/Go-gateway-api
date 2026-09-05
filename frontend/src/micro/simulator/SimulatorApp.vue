<script setup>
import { ref, onMounted, computed, onBeforeUnmount } from 'vue'
import { api, auth } from '../../lib/api.js'
import { show } from '../../lib/toast.js'

const props = defineProps({ init: { type: String, default: '' } })

const apis = ref([])
const selectedId = ref(props.init || '')
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

// group: apis with /api/v1 prefix = "System", rest = "Custom"
const groupedApis = computed(() => {
  const list = filteredApis.value
  const system = list.filter((a) => a.base_path.startsWith('/api/'))
  const custom = list.filter((a) => !a.base_path.startsWith('/api/'))
  return [
    ...(system.length ? [{ type: 'header', label: 'System API' }, ...system] : []),
    ...(custom.length ? [{ type: 'header', label: 'Custom API' }, ...custom] : []),
  ]
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
const inspectTab = ref('body') // 'body' | 'headers' | 'request'
const tokenText = ref('')
const tokenCopied = ref(false)
const allMethodBody = ['POST', 'PUT', 'PATCH']

onMounted(async () => {
  document.addEventListener('mousedown', onClickOutside)
  try {
    apis.value = await api.listApis()
    if (!selectedId.value) selectedId.value = String(apis.value[0]?.id || '')
    if (selectedId.value) { autoSend() }
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

// URL untuk curl: pakai path relatif saja (tanpa origin dev server)
function buildCurlPath() {
  const base = selected.value.base_path.replace(/\/+$/, '')
  const sub = (subpath.value || '').replace(/^\/+/, '')
  return base + (sub ? '/' + sub : '')
}

async function send() {
  if (!selectedId.value) return
  sending.value = true
  resp.value = null
  inspectTab.value = 'body'
  const url = buildUrl()
  const reqHeaders = { Accept: 'application/json' }
  if (allMethodBody.includes(method.value)) reqHeaders['Content-Type'] = 'application/json'
  if (attachJwt.value && auth.token) reqHeaders['Authorization'] = 'Bearer ' + auth.token
  const started = performance.now()
  const ctrl = new AbortController()
  const t = setTimeout(() => ctrl.abort(), 10000)
  try {
    const r = await fetch(url, { method: method.value, headers: reqHeaders, body: allMethodBody.includes(method.value) && bodyText.value ? bodyText.value : undefined, signal: ctrl.signal })
    const text = await r.text()
    const resHeaders = {}
    r.headers.forEach((v, k) => { resHeaders[k] = v })
    resp.value = {
      ok: r.ok,
      status: r.status,
      statusText: r.statusText,
      ms: Math.round(performance.now() - started),
      size: text.length,
      text,
      url,
      method: method.value,
      reqHeaders,
      resHeaders,
    }
  } catch (e) {
    resp.value = {
      ok: false, status: 0, statusText: '',
      ms: Math.round(performance.now() - started), size: 0,
      text: e.name === 'AbortError' ? 'Request timed out after 10s' : `Gagal mengirim: ${e.message}`,
      url, method: method.value, reqHeaders, resHeaders: {},
    }
  } finally {
    clearTimeout(t)
    sending.value = false
  }
}

const curlCopied = ref(false)

const curlLive = computed(() => {
  if (!selected.value) return ''
  const path = buildCurlPath()
  const lines = [`curl -s -X ${method.value} "https://your-gateway${path}"`]
  if (attachJwt.value) lines.push(`  -H "Authorization: Bearer $TOKEN"`)
  if (allMethodBody.includes(method.value)) {
    lines.push(`  -H "Content-Type: application/json"`)
    if (bodyText.value) lines.push(`  -d '${bodyText.value.replace(/'/g, `'\\''`)}'`)
  }
  return lines.join(' \\\n')
})

// legacy full doc (with token acquisition step)
const curlDoc = computed(() => {
  if (!selected.value) return ''
  const path = buildCurlPath()
  const methodFlag = method.value !== 'GET' ? ` -X ${method.value}` : ''
  const jwtHeader = attachJwt.value ? `\n  -H "Authorization: Bearer $TOKEN"` : ''
  const ctHeader = allMethodBody.includes(method.value) ? `\n  -H "Content-Type: application/json"` : ''
  const body = allMethodBody.includes(method.value) && bodyText.value
    ? `\n  -d '${bodyText.value.replace(/'/g, `'\\''`)}'`
    : ''

  return `# Ganti GATEWAY_URL dengan URL gateway kamu
GATEWAY_URL="https://your-gateway"

# Ambil token
TOKEN=$(curl -s -X POST "$GATEWAY_URL/api/v1/auth/login" \\
  -H "Content-Type: application/json" \\
  -d '{"email":"admin@example.com","password":"admin123"}' \\
  | jq -r .data.access_token)

# Kirim request
curl -s${methodFlag} "$GATEWAY_URL${path}"${jwtHeader}${ctHeader}${body}`
})

function copyCurl() {
  const text = curlDoc.value
  if (!text) return
  navigator.clipboard?.writeText(text).then(() => {
    curlCopied.value = true
    setTimeout(() => (curlCopied.value = false), 1800)
  }).catch(() => {})
  show('Curl command disalin!')
}

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
            <ul class="thin-scroll max-h-[240px] overflow-y-auto py-1">
              <li v-if="filteredApis.length === 0" class="px-4 py-5 text-center text-[12px] text-mute">
                Tidak ada hasil
              </li>
              <template v-for="item in groupedApis" :key="item.type === 'header' ? 'h-' + item.label : item.id">
                <!-- section header -->
                <li v-if="item.type === 'header'" class="px-3 pb-0.5 pt-3 first:pt-2">
                  <span class="label-sm">{{ item.label }}</span>
                </li>
                <!-- api item: single clean row -->
                <li v-else
                  class="tappable flex cursor-pointer items-center gap-2.5 rounded-xl mx-1.5 px-2.5 py-2 transition-colors hover:bg-panel-2"
                  :class="{ 'bg-[var(--color-accent-tint)]': String(item.id) === String(selectedId) }"
                  @click="pickApi(item.id)"
                >
                  <!-- active dot / empty spacer -->
                  <span class="h-1.5 w-1.5 shrink-0 rounded-full transition-colors"
                    :class="String(item.id) === String(selectedId) ? 'bg-[var(--color-accent)]' : 'bg-transparent'">
                  </span>
                  <!-- name -->
                  <span class="min-w-0 flex-1 truncate text-[13px]"
                    :class="String(item.id) === String(selectedId) ? 'font-semibold text-ink' : 'text-ink'">
                    {{ item.name }}
                  </span>
                  <!-- path pill -->
                  <code class="shrink-0 truncate rounded-md bg-panel-2 px-1.5 py-0.5 text-[10px] text-mute max-w-[120px]">{{ item.base_path }}</code>
                  <!-- auth dot -->
                  <span class="h-1.5 w-1.5 shrink-0 rounded-full"
                    :class="item.requires_auth ? 'bg-[#0071e3]' : 'bg-ok'">
                  </span>
                </li>
              </template>
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
          <div class="label-sm">Proteksi</div>
          <div class="pt-0.5">{{ selected?.requires_auth ? 'JWT · auth_request' : 'Terbuka' }}</div>
        </div>
        <div class="rounded-xl bg-panel-2 px-3 py-2">
          <div class="label-sm">Methods</div>
          <div class="break-all pt-0.5">{{ selected?.methods.join(', ') }}</div>
        </div>
        <div class="rounded-xl bg-panel-2 px-3 py-2">
          <div class="label-sm">Rate / Status</div>
          <div class="pt-0.5">{{ selected?.rate_limit_rpm }}r/m · {{ selected?.status }}</div>
        </div>
      </div>

      <div class="mt-3 flex flex-col gap-2 min-[400px]:flex-row">
        <button class="btn-primary tappable flex-1" :disabled="publishing" @click="publish">
          {{ publishing ? 'Menerbitkan…' : 'Terapkan & reload' }}
        </button>
      </div>
    </div>

    <!-- ── send a test request ── -->
    <div class="mb-4 rounded-[18px] bg-panel p-4 shadow-sm">
      <div class="label-sm pb-2">
        Kirim request
        <span class="text-mute">
          {{ selected?.requires_auth ? '· perlu JWT' : '· tanpa autentikasi' }}
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
          Sertakan JWT
        </label>
        <button class="btn-ghost tappable !py-2 text-[12px] ml-auto" @click="grabToken">
          {{ tokenCopied ? 'Tersalin ✓' : tokenText ? 'Perbarui token' : 'Ambil token' }}
        </button>
        <button class="btn-primary tappable w-full min-[400px]:w-auto"
          :disabled="sending || !selectedId" @click="send">
          {{ sending ? 'Mengirim…' : '▶ Kirim' }}
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

      <!-- inspect panel -->
      <div v-if="resp" class="mt-3 overflow-hidden rounded-xl border" :style="{ borderColor: 'var(--color-line)' }">
        <!-- status bar -->
        <div class="flex flex-wrap items-center gap-x-2 gap-y-1 border-b px-3 py-2 text-[11px]"
          :style="{ borderColor: 'var(--color-line)' }">
          <span class="shrink-0 rounded-full px-2 py-0.5 font-bold"
            :class="resp.status === 0 ? 'bg-panel-2 text-mute' : resp.status < 400 ? 'bg-ok/10 text-ok' : 'bg-bad/10 text-bad'">
            {{ resp.status || 'ERR' }}{{ resp.statusText ? ' ' + resp.statusText : '' }}
          </span>
          <span class="shrink-0 text-mute">{{ resp.ms }}ms</span>
          <span class="shrink-0 text-mute">{{ resp.size }}B</span>
          <span class="w-full truncate text-mute">{{ resp.method }} {{ resp.url }}</span>
        </div>

        <!-- tab bar -->
        <div class="flex border-b text-[11px]" :style="{ borderColor: 'var(--color-line)' }">
          <button v-for="tab in ['body', 'headers', 'request']" :key="tab"
            class="tappable px-3 py-2 font-medium capitalize transition-colors"
            :class="inspectTab === tab
              ? 'border-b-2 border-[var(--color-accent)] text-[var(--color-accent)]'
              : 'text-mute hover:text-ink'"
            :style="inspectTab === tab ? { marginBottom: '-1px' } : {}"
            @click="inspectTab = tab">
            {{ tab === 'body' ? 'Body' : tab === 'headers' ? 'Res Headers' : 'Req Headers' }}
          </button>
        </div>

        <!-- body tab -->
        <pre v-if="inspectTab === 'body'"
          class="thin-scroll m-0 max-h-[280px] overflow-auto px-3 py-2.5 text-[11px] leading-relaxed">{{ pretty(resp.text) || '(body kosong)' }}</pre>

        <!-- response headers tab -->
        <div v-else-if="inspectTab === 'headers'"
          class="thin-scroll max-h-[280px] overflow-auto divide-y"
          :style="{ borderColor: 'var(--color-line)' }">
          <div v-for="(val, key) in resp.resHeaders" :key="key"
            class="flex gap-2 px-3 py-1.5 text-[11px]">
            <span class="w-[40%] shrink-0 font-medium text-[var(--color-accent)] truncate">{{ key }}</span>
            <span class="min-w-0 flex-1 break-all text-mute">{{ val }}</span>
          </div>
          <div v-if="!Object.keys(resp.resHeaders).length"
            class="px-3 py-6 text-center text-[11px] text-mute">Tidak ada header</div>
        </div>

        <!-- request headers tab -->
        <div v-else
          class="thin-scroll max-h-[280px] overflow-auto divide-y"
          :style="{ borderColor: 'var(--color-line)' }">
          <div v-for="(val, key) in resp.reqHeaders" :key="key"
            class="flex gap-2 px-3 py-1.5 text-[11px]">
            <span class="w-[40%] shrink-0 font-medium text-[var(--color-accent)] truncate">{{ key }}</span>
            <span class="min-w-0 flex-1 break-all text-mute">{{ val }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ── curl documentation ── -->
    <div class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
      <div class="flex items-center gap-2 px-4 py-3">
        <span class="h-2 w-2 rounded-full bg-ok shrink-0"></span>
        <span class="flex-1 text-[11px] text-mute">Contoh curl untuk API ini</span>
        <button
          v-if="curlDoc"
          class="tappable flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-[11px] font-medium transition-colors"
          :class="curlCopied ? 'bg-ok/10 text-ok' : 'bg-panel-2 text-mute hover:text-ink'"
          @click="copyCurl"
        >
          <!-- copy icon -->
          <svg v-if="!curlCopied" width="12" height="12" viewBox="0 0 14 14" fill="none">
            <rect x="4.5" y="1.5" width="8" height="9" rx="1.5" stroke="currentColor" stroke-width="1.4"/>
            <path d="M1.5 4.5H3a1 1 0 011 1v7a1 1 0 001 1h5.5" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
          </svg>
          <svg v-else width="12" height="12" viewBox="0 0 14 14" fill="none">
            <path d="M2.5 7l3.5 3.5 5.5-6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          {{ curlCopied ? 'Tersalin!' : 'Salin' }}
        </button>
      </div>
      <pre v-if="curlDoc" class="thin-scroll m-0 max-h-[260px] overflow-x-auto border-t bg-panel-2 px-4 py-3 text-[11px] leading-relaxed"
        :style="{ borderColor: 'var(--color-line)' }">{{ curlDoc }}</pre>
      <div v-else class="border-t px-4 py-10 text-center text-mute"
        :style="{ borderColor: 'var(--color-line)' }">
        Pilih API untuk melihat contoh curl.
      </div>
    </div>

  </div>
</template>
