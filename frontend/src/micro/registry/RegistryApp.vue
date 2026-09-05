<script setup>
import { ref, onMounted, watch, computed, onUnmounted } from 'vue'
import { api } from '../../lib/api.js'
import { show } from '../../lib/toast.js'

const props = defineProps({ init: { type: String, default: '' } })

const apis = ref([])
const loading = ref(true)
const error = ref('')

const perPage = ref(5)
const mobile = ref(window.innerWidth < 640)
function onResize() { mobile.value = window.innerWidth < 640; perPage.value = mobile.value ? 3 : 5 }
onMounted(() => window.addEventListener('resize', onResize))
onUnmounted(() => window.removeEventListener('resize', onResize))
const page = ref(1)
const totalPages = computed(() => Math.max(1, Math.ceil(apis.value.length / perPage)))
watch(totalPages, (tp) => { if (page.value > tp) page.value = tp })
const pagedApis = computed(() => apis.value.slice((page.value - 1) * perPage, page.value * perPage))

const mode = ref(props.init ? 'edit' : 'list')
const editId = ref(props.init ? Number(props.init) : null)

const form = ref({ name: '', base_path: '', upstream: '', methods: ['GET'], requires_auth: true, rate_limit_rpm: 60, is_active: true, note: '' })
const allMethods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']
const busy = ref(false)

onMounted(async () => {
  try {
    apis.value = await api.listApis()
    if (editId.value) await openEdit(editId.value)
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
})

async function openEdit(id) {
  try {
    const a = await api.getApi(id)
    form.value = { name: a.name, base_path: a.base_path, upstream: a.upstream, methods: a.methods, requires_auth: a.requires_auth, rate_limit_rpm: a.rate_limit_rpm, is_active: a.is_active, note: a.note || '' }
    editId.value = id
    mode.value = 'edit'
  } catch (e) {
    show(e.message, 'err')
  }
}

function newForm() {
  form.value = { name: '', base_path: '', upstream: '', methods: ['GET'], requires_auth: true, rate_limit_rpm: 60, is_active: true, note: '' }
  editId.value = null
  mode.value = 'edit'
}

function back() {
  mode.value = 'list'
  load()
}

async function load() {
  page.value = 1
  try {
    apis.value = await api.listApis()
  } catch (e) {
    error.value = e.message
  }
}

async function submit() {
  busy.value = true
  try {
    const payload = { ...form.value, methods: form.value.methods.length ? form.value.methods : ['GET'] }
    if (editId.value) {
      await api.updateApi(editId.value, payload)
      show('Registered API updated · gateway reloaded')
    } else {
      const c = await api.createApi(payload)
      show(`"${c.base_path}" registered · now live`)
    }
    back()
  } catch (e) {
    show(e.message, 'err')
  } finally {
    busy.value = false
  }
}

async function remove(a) {
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
  ({ healthy: 'bg-ok/10 text-ok', unhealthy: 'bg-bad/10 text-bad' })[s] || 'bg-panel-2 text-mute'

function rowLabel(a) {
  const n = a.methods.length
  const first = a.methods[0] || 'GET'
  return n > 1 ? `${first}+${n - 1}` : first
}
</script>

<template>
  <div>
    <!-- list mode -->
    <div v-if="mode === 'list'">
      <header class="mb-4 flex items-center gap-3">
        <div class="min-w-0">
          <h1 class="m-0 text-[20px] font-bold tracking-tight">API Registry</h1>
          <p class="label-sm m-0 mt-0.5 truncate">{{ pagedApis.length }} shown · {{ apis.length }} total · {{ apis.filter((a) => a.is_active).length }} active</p>
        </div>
        <span class="flex-1"></span>
        <button class="btn-primary tappable !px-3 !py-2 text-[13px]" @click="newForm">+ Register</button>
      </header>

      <div v-if="error" class="mb-3 text-[13px] text-bad">{{ error }}</div>
      <div v-if="loading" class="flex flex-col gap-2">
        <div v-for="i in 4" :key="i" class="animate-pulse rounded-[18px] bg-panel shadow-[0_1px_2px_rgba(0,0,0,0.04)]">
          <div class="px-4 py-3">
            <div class="h-3.5 w-32 rounded-md bg-panel-2"></div>
            <div class="mt-2 h-3 w-48 rounded-md bg-panel-2"></div>
          </div>
          <div class="flex items-center gap-2 border-t px-3 py-2" :style="{ borderColor: 'var(--color-line)' }">
            <div class="h-3 w-20 rounded-md bg-panel-2"></div>
            <span class="flex-1"></span>
            <div class="h-5 w-12 rounded-full bg-panel-2"></div>
          </div>
        </div>
      </div>

      <!-- empty state -->
      <div v-else-if="!apis.length" class="rounded-[18px] bg-panel py-14 text-center shadow-sm">
        <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-panel-2">
          <svg width="26" height="26" viewBox="0 0 24 24" fill="none" class="text-mute">
            <rect x="3" y="3" width="18" height="18" rx="3" stroke="currentColor" stroke-width="1.8"/>
            <path d="M3 9h18M9 9v12" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
          </svg>
        </div>
        <p class="mb-0.5 text-[15px] font-semibold">Belum ada API terdaftar</p>
        <p class="mb-5 text-[13px] text-mute">Gateway hanya mengekspos API yang ada di registry ini.</p>
        <button class="btn-primary tappable" @click="newForm">+ Register API pertama</button>
      </div>

      <div v-else class="flex flex-col gap-2">
        <div v-for="a in pagedApis" :key="a.id" class="rounded-[18px] bg-panel shadow-[0_1px_2px_rgba(0,0,0,0.04)]">
          <button class="inset-cell tappable w-full text-left" style="border-radius: 18px; border-bottom: 1px solid transparent" @click="openEdit(a.id)">
            <div class="min-w-0">
              <div class="truncate font-[600]">{{ a.name }}</div>
              <div class="mt-0.5 flex items-center gap-1.5 text-[12px] text-mute">
                <code class="text-accent">{{ a.base_path }}/*</code>
                <span>·</span>
                <span>{{ a.upstream.replace(/^https?:\/\//, '') }}</span>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <span v-if="a.requires_auth" class="rounded-full bg-accent-tint px-2 py-0.5 text-[10px] font-bold text-accent-2">JWT</span>
              <span v-else class="rounded-full bg-panel-2 px-2 py-0.5 text-[10px] font-bold text-mute">OPEN</span>
              <span class="rounded-full px-2 py-0.5 text-[10px] font-bold" :class="statusClass(a.status)">{{ a.status }}</span>
              <svg width="15" height="15" viewBox="0 0 24 24" class="text-mute"><path fill="none" stroke="currentColor" stroke-width="2.4" d="M9 6l6 6-6 6"/></svg>
            </div>
          </button>
          <div class="flex items-center gap-2 border-t px-3 py-2" :style="{ borderColor: 'var(--color-line)' }">
            <span class="text-[11px] text-mute">{{ rowLabel(a) }} · {{ a.rate_limit_rpm }}r/m</span>
            <span class="flex-1"></span>
            <span v-if="!a.deleted_at" class="rounded-full px-2 py-0.5 text-[10px] font-bold" :class="a.is_active ? 'bg-ok/10 text-ok' : 'bg-panel-2 text-mute'">
              {{ a.is_active ? 'ON' : 'OFF' }}
            </span>
            <button class="tappable text-[12px] font-medium text-bad" @click="remove(a)">Delete</button>
            <button v-if="a.deleted_at" class="tappable text-[12px] font-medium text-ok" @click="restore(a.id)">Restore</button>
          </div>
        </div>
      </div>

      <div v-if="totalPages > 1" class="mt-3 flex items-center justify-between rounded-[18px] bg-panel px-3 py-2 shadow-sm">
        <button class="tappable flex h-9 w-9 items-center justify-center rounded-full text-ink disabled:opacity-30"
          :disabled="page <= 1" @click="page--">
          <svg width="17" height="17" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-width="2.2" d="M15 6l-6 6 6 6"/></svg>
        </button>
        <div class="flex items-center gap-1.5">
          <button v-for="p in totalPages" :key="p" class="tappable h-9 min-w-9 rounded-full px-2 text-[13px] font-semibold"
            :class="p === page ? 'bg-accent text-white' : 'text-mute hover:bg-panel-2'" @click="page = p">{{ p }}</button>
        </div>        <button class="tappable flex h-9 w-9 items-center justify-center rounded-full text-ink disabled:opacity-30"
          :disabled="page >= totalPages" @click="page++">
          <svg width="17" height="17" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-width="2.2" d="M9 6l6 6-6 6"/></svg>
        </button>
      </div>
    </div>

    <!-- form mode -->
    <div v-else class="mx-auto max-w-xl">
      <header class="mb-4 flex items-center gap-3">
        <button class="tappable flex h-9 w-9 items-center justify-center rounded-full bg-panel shadow-sm" @click="back">
          <svg width="18" height="18" viewBox="0 0 24 24" class="text-ink"><path fill="none" stroke="currentColor" stroke-width="2.4" d="M15 6l-6 6 6 6"/></svg>
        </button>
        <h1 class="m-0 text-[20px] font-bold tracking-tight">{{ editId ? 'Edit API' : 'Register API' }}</h1>
      </header>

      <div class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
        <div class="inset-cell">
          <span class="label-sm w-28 shrink-0">Name</span>
          <input v-model.trim="form.name" type="text" required placeholder="Order Service"
            class="w-full bg-transparent px-0 text-right text-[14px] placeholder:text-mute/60 focus:outline-none" />
        </div>
        <div class="inset-cell">
          <span class="label-sm w-28 shrink-0">Base path</span>
          <input v-model.trim="form.base_path" type="text" required placeholder="/orders" pattern="/[a-zA-Z0-9._~-]+"
            class="w-full bg-transparent px-0 text-right text-[14px] placeholder:text-mute/60 focus:outline-none" />
        </div>
        <div class="inset-cell">
          <span class="label-sm w-28 shrink-0">Upstream</span>
          <input v-model.trim="form.upstream" type="text" required placeholder="http://svc:8080"
            class="w-full bg-transparent px-0 text-right text-[14px] placeholder:text-mute/60 focus:outline-none" />
        </div>
        <div class="px-4 py-3">
          <span class="label-sm block pb-2">Methods</span>
          <div class="seg flex-wrap">
            <button v-for="m in allMethods" :key="m" type="button" :class="{ on: form.methods.includes(m) }" @click="form.methods.includes(m) ? form.methods.splice(form.methods.indexOf(m), 1) : form.methods.push(m)">
              {{ m }}
            </button>
          </div>
        </div>
        <div class="inset-cell">
          <span class="label-sm shrink-0">Rate limit /min</span>
          <input v-model.number="form.rate_limit_rpm" type="number" min="1" max="100000" required
            class="w-28 bg-transparent text-right text-[14px] focus:outline-none" />
        </div>
        <div class="inset-cell">
          <span class="label-sm shrink-0">Note</span>
          <input v-model.trim="form.note" type="text" placeholder="optional"
            class="w-full bg-transparent px-0 text-right text-[14px] placeholder:text-mute/60 focus:outline-none" />
        </div>
        <div class="px-4 py-3">
          <label class="flex items-center justify-between py-1.5">
            <span class="text-[14px]">Require JWT</span>
            <span class="relative inline-flex h-[30px] w-[50px] rounded-full transition" :class="form.requires_auth ? 'bg-accent' : 'bg-panel-2'" style="border: 1px solid var(--color-line)">
              <input v-model="form.requires_auth" type="checkbox" class="peer sr-only" />
              <span class="absolute top-[1px] h-6 w-6 rounded-full bg-white shadow transition-all" :class="form.requires_auth ? 'left-[23px]' : 'left-[1px]'"></span>
            </span>
          </label>
          <label class="flex items-center justify-between py-1.5">
            <span class="text-[14px]">Active</span>
            <span class="relative inline-flex h-[30px] w-[50px] rounded-full transition" :class="form.is_active ? 'bg-accent' : 'bg-panel-2'" style="border: 1px solid var(--color-line)">
              <input v-model="form.is_active" type="checkbox" class="sr-only" />
              <span class="absolute top-[1px] h-6 w-6 rounded-full bg-white shadow transition-all" :class="form.is_active ? 'left-[23px]' : 'left-[1px]'"></span>
            </span>
          </label>
        </div>
        <div class="flex gap-2.5 border-t px-4 py-3" :style="{ borderColor: 'var(--color-line)' }">
          <button class="btn-primary tappable flex-1" :disabled="busy" @click="submit">
            {{ busy ? 'Saving…' : editId ? 'Save & re-publish' : 'Register & expose' }}
          </button>
          <button class="btn-ghost tappable" @click="back">Cancel</button>
        </div>
      </div>
    </div>
  </div>
</template>