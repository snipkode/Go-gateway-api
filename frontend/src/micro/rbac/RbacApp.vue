<script setup>
import { ref, computed, watch } from 'vue'
import { api } from '../../lib/api.js'
import { show } from '../../lib/toast.js'

// ── tabs ──
const TABS = [
  { key: 'users',       label: 'Users' },
  { key: 'roles',       label: 'Roles' },
  { key: 'permissions', label: 'Permissions' },
  { key: 'ratelimits',  label: 'Rate Limits' },
  { key: 'audit',       label: 'Audit' },
]
const tab = ref('users')

function fmtDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleString('id-ID', { dateStyle: 'short', timeStyle: 'short' })
}

function prettyJson(v) {
  if (!v) return '—'
  try { return JSON.stringify(typeof v === 'string' ? JSON.parse(v) : v, null, 2) } catch { return String(v) }
}

// ═══════════════════════════════════════════════
// USERS
// field: id, email, name, status, provider, created_at, deleted_at
// create: { email, name, password }
// update: { name, status }
// ═══════════════════════════════════════════════
const users      = ref([])
const usersLoading = ref(false)
const userForm   = ref(null)
const userBusy   = ref(false)
const userSearch = ref('')

const filteredUsers = computed(() => {
  const q = userSearch.value.toLowerCase()
  return users.value.filter(u =>
    !q || u.email?.toLowerCase().includes(q) || u.name?.toLowerCase().includes(q)
  )
})

async function loadUsers() {
  usersLoading.value = true
  try { users.value = await api.listUsers() }
  catch (e) { show(e.message) }
  finally { usersLoading.value = false }
}

function openUserForm(u = null) {
  userForm.value = u
    ? { _mode: 'edit', id: u.id, name: u.name || '', status: u.status || 'active', password: '' }
    : { _mode: 'create', name: '', email: '', password: '' }
}

async function submitUser() {
  userBusy.value = true
  try {
    const f = userForm.value
    if (f._mode === 'edit') {
      const payload = { name: f.name, status: f.status }
      if (f.password) payload.password = f.password
      await api.updateUser(f.id, payload)
      show('User diperbarui')
    } else {
      await api.createUser({ email: f.email, name: f.name, password: f.password })
      show('User dibuat')
    }
    userForm.value = null
    loadUsers()
  } catch (e) { show(e.message) }
  finally { userBusy.value = false }
}

async function deleteUser(u) {
  if (!confirm(`Hapus ${u.email}?`)) return
  try { await api.deleteUser(u.id); show('User dihapus'); loadUsers() }
  catch (e) { show(e.message) }
}

async function restoreUser(u) {
  try { await api.restoreUser(u.id); show('User dipulihkan'); loadUsers() }
  catch (e) { show(e.message) }
}

// ═══════════════════════════════════════════════
// ROLES
// field: id, name, slug, description, deleted_at
// permissions loaded separately via getRole
// ═══════════════════════════════════════════════
const roles        = ref([])
const allPerms     = ref([])
const rolesLoading = ref(false)
const roleForm     = ref(null)
const roleBusy     = ref(false)
const roleDetail   = ref(null)  // { role, grantedIds }

async function loadRoles() {
  rolesLoading.value = true
  try {
    [roles.value, allPerms.value] = await Promise.all([
      api.listRoles(),
      api.listPermissions(),
    ])
  } catch (e) { show(e.message) }
  finally { rolesLoading.value = false }
}

async function openRoleDetail(r) {
  try {
    const full = await api.getRole(r.id)
    const grantedIds = new Set((full.permissions ?? []).map(p => p.id))
    roleDetail.value = { role: full, grantedIds }
    if (!allPerms.value.length) allPerms.value = await api.listPermissions()
  } catch (e) { show(e.message) }
}

function openRoleForm(r = null) {
  roleForm.value = r
    ? { id: r.id, name: r.name, slug: r.slug, description: r.description || '' }
    : { id: null, name: '', slug: '', description: '' }
}

async function submitRole() {
  roleBusy.value = true
  try {
    const { id, ...rest } = roleForm.value
    if (id) { await api.updateRole(id, rest); show('Role diperbarui') }
    else { await api.createRole(rest); show('Role dibuat') }
    roleForm.value = null
    loadRoles()
  } catch (e) { show(e.message) }
  finally { roleBusy.value = false }
}

async function deleteRole(r) {
  if (!confirm(`Hapus role "${r.name}"?`)) return
  try { await api.deleteRole(r.id); show('Role dihapus'); loadRoles() }
  catch (e) { show(e.message) }
}

async function restoreRole(r) {
  try { await api.restoreRole(r.id); show('Role dipulihkan'); loadRoles() }
  catch (e) { show(e.message) }
}

async function togglePermission(perm) {
  const rd = roleDetail.value
  if (!rd) return
  try {
    if (rd.grantedIds.has(perm.id)) {
      await api.revokePermission(rd.role.id, perm.id)
      rd.grantedIds.delete(perm.id)
      show(`${perm.slug} dicabut`)
    } else {
      await api.grantPermission(rd.role.id, perm.id)
      rd.grantedIds.add(perm.id)
      show(`${perm.slug} diberikan`)
    }
  } catch (e) { show(e.message) }
}

// ═══════════════════════════════════════════════
// PERMISSIONS
// field: id, name, slug, description, deleted_at
// ═══════════════════════════════════════════════
const permissions  = ref([])
const permsLoading = ref(false)
const permForm     = ref(null)
const permBusy     = ref(false)

async function loadPermissions() {
  permsLoading.value = true
  try { permissions.value = await api.listPermissions() }
  catch (e) { show(e.message) }
  finally { permsLoading.value = false }
}

function openPermForm(p = null) {
  permForm.value = p
    ? { id: p.id, name: p.name, slug: p.slug, description: p.description || '' }
    : { id: null, name: '', slug: '', description: '' }
}

async function submitPerm() {
  permBusy.value = true
  try {
    const { id, ...rest } = permForm.value
    if (id) { await api.updatePermission(id, rest); show('Permission diperbarui') }
    else { await api.createPermission(rest); show('Permission dibuat') }
    permForm.value = null
    loadPermissions()
  } catch (e) { show(e.message) }
  finally { permBusy.value = false }
}

async function deletePerm(p) {
  if (!confirm(`Hapus "${p.slug}"?`)) return
  try { await api.deletePermission(p.id); show('Permission dihapus'); loadPermissions() }
  catch (e) { show(e.message) }
}

async function restorePerm(p) {
  try { await api.restorePermission(p.id); show('Permission dipulihkan'); loadPermissions() }
  catch (e) { show(e.message) }
}

// ═══════════════════════════════════════════════
// RATE LIMITS
// field: id, name, scope, identifier, requests, window_seconds, enabled, priority
// ═══════════════════════════════════════════════
const rateLimits = ref([])
const rlLoading  = ref(false)
const rlForm     = ref(null)
const rlBusy     = ref(false)
const SCOPES     = ['global', 'ip', 'user', 'role', 'route', 'api_key']

async function loadRateLimits() {
  rlLoading.value = true
  try { rateLimits.value = await api.listRateLimits() }
  catch (e) { show(e.message) }
  finally { rlLoading.value = false }
}

function openRlForm(r = null) {
  rlForm.value = r
    ? { id: r.id, name: r.name, scope: r.scope, identifier: r.identifier || '',
        requests: r.requests, window_seconds: r.window_seconds,
        enabled: r.enabled ?? true, priority: r.priority ?? 0 }
    : { id: null, name: '', scope: 'ip', identifier: '',
        requests: 60, window_seconds: 60, enabled: true, priority: 0 }
}

async function submitRl() {
  rlBusy.value = true
  try {
    const { id, ...rest } = rlForm.value
    if (id) { await api.updateRateLimit(id, rest); show('Rate limit diperbarui') }
    else { await api.createRateLimit(rest); show('Rate limit dibuat') }
    rlForm.value = null
    loadRateLimits()
  } catch (e) { show(e.message) }
  finally { rlBusy.value = false }
}

async function deleteRl(r) {
  if (!confirm(`Hapus rule "${r.name}"?`)) return
  try { await api.deleteRateLimit(r.id); show('Rate limit dihapus'); loadRateLimits() }
  catch (e) { show(e.message) }
}

async function restoreRl(r) {
  try { await api.restoreRateLimit(r.id); show('Rate limit dipulihkan'); loadRateLimits() }
  catch (e) { show(e.message) }
}

// ═══════════════════════════════════════════════
// AUDIT LOGS
// field: id, action, resource, resource_id, user_id, ip_address,
//        method, path, old_data(obj), new_data(obj), created_at
// ═══════════════════════════════════════════════
const auditLogs    = ref([])
const auditLoading = ref(false)
const auditSearch  = ref('')
const auditResource = ref('')
const auditPage    = ref(1)
const AUDIT_PER    = 15
const auditDetail  = ref(null)

const filteredAudit = computed(() => {
  const q = auditSearch.value.toLowerCase()
  return auditLogs.value.filter(l =>
    !q ||
    l.action?.toLowerCase().includes(q) ||
    l.resource?.toLowerCase().includes(q) ||
    l.path?.toLowerCase().includes(q)
  )
})

const auditPages = computed(() => Math.max(1, Math.ceil(filteredAudit.value.length / AUDIT_PER)))
const auditPaged = computed(() => {
  const s = (auditPage.value - 1) * AUDIT_PER
  return filteredAudit.value.slice(s, s + AUDIT_PER)
})
watch(filteredAudit, () => { auditPage.value = 1 })

async function loadAudit() {
  auditLoading.value = true
  auditDetail.value = null
  try {
    const params = {}
    if (auditResource.value) params.resource = auditResource.value
    auditLogs.value = await api.listAuditLogs(params)
  } catch (e) { show(e.message) }
  finally { auditLoading.value = false }
}

function actionColor(a) {
  return {
    create:  'bg-ok/10 text-ok',
    restore: 'bg-ok/10 text-ok',
    delete:  'bg-bad/10 text-bad',
    update:  'bg-warn/10 text-warn',
    login:   'bg-[#0071e3]/10 text-[#0071e3]',
    logout:  'bg-panel-2 text-mute',
  }[a] ?? 'bg-panel-2 text-mute'
}

// ── load on tab switch ──
watch(tab, (t) => {
  if (t === 'users'       && !users.value.length)      loadUsers()
  if (t === 'roles'       && !roles.value.length)       loadRoles()
  if (t === 'permissions' && !permissions.value.length) loadPermissions()
  if (t === 'ratelimits'  && !rateLimits.value.length)  loadRateLimits()
  if (t === 'audit'       && !auditLogs.value.length)   loadAudit()
}, { immediate: true })

// shared back-button svg
const BACK_SVG = `<svg width="18" height="18" viewBox="0 0 24 24" fill="none"><path d="M15 6l-6 6 6 6" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/></svg>`
</script>

<template>
  <div>
    <header class="mb-4">
      <h1 class="m-0 text-[20px] font-bold tracking-tight">RBAC & Admin</h1>
      <p class="label-sm m-0 mt-0.5">Users · Roles · Permissions · Rate Limits · Audit</p>
    </header>

    <!-- tab bar -->
    <div class="thin-scroll mb-4 flex gap-1 overflow-x-auto rounded-[14px] bg-panel-2 p-1">
      <button v-for="t in TABS" :key="t.key"
        class="tappable shrink-0 rounded-[10px] px-3 py-1.5 text-[13px] font-medium transition-colors"
        :class="tab === t.key ? 'bg-panel text-ink shadow-sm' : 'text-mute hover:text-ink'"
        @click="tab = t.key">
        {{ t.label }}
      </button>
    </div>

    <!-- ══ USERS ══ -->
    <template v-if="tab === 'users'">
      <template v-if="!userForm">
        <div class="mb-3 flex gap-2">
          <input v-model="userSearch" type="text" placeholder="Cari nama atau email…"
            class="flex-1 rounded-xl border bg-panel px-3 py-2 text-[13px] focus:outline-none"
            :style="{ borderColor: 'var(--color-line)' }" />
          <button class="btn-primary tappable !px-3 !py-2 text-[13px]" @click="openUserForm()">+ Tambah</button>
        </div>

        <div v-if="usersLoading" class="space-y-2">
          <div v-for="i in 4" :key="i" class="h-[58px] animate-pulse rounded-[18px] bg-panel"></div>
        </div>
        <div v-else-if="!filteredUsers.length" class="rounded-[18px] bg-panel py-14 text-center shadow-sm">
          <div class="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-panel-2">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none"><path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2M12 11a4 4 0 100-8 4 4 0 000 8z" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/></svg>
          </div>
          <p class="mb-0.5 text-[14px] font-semibold">Belum ada user</p>
          <p class="mb-4 text-[12px] text-mute">{{ userSearch ? 'Tidak ditemukan' : 'Tambahkan user pertama' }}</p>
          <button v-if="!userSearch" class="btn-primary tappable" @click="openUserForm()">+ Tambah User</button>
        </div>
        <div v-else class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div v-for="u in filteredUsers" :key="u.id" class="inset-cell">
            <div class="flex min-w-0 flex-1 items-center gap-2.5">
              <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent-tint)] text-[12px] font-bold text-accent">
                {{ (u.name || u.email || '?')[0].toUpperCase() }}
              </div>
              <div class="min-w-0">
                <div class="truncate text-[13px] font-medium">{{ u.name || '—' }}</div>
                <div class="truncate text-[11px] text-mute">{{ u.email }}
                  <span v-if="u.provider && u.provider !== 'local'" class="ml-1 rounded-full bg-panel-2 px-1.5 py-0.5 text-[9px]">{{ u.provider }}</span>
                </div>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <span class="rounded-full px-1.5 py-0.5 text-[10px] font-bold"
                :class="u.deleted_at ? 'bg-bad/10 text-bad' : u.status === 'active' ? 'bg-ok/10 text-ok' : 'bg-panel-2 text-mute'">
                {{ u.deleted_at ? 'deleted' : u.status }}
              </span>
              <button class="tappable text-[12px] text-accent" @click="openUserForm(u)">Edit</button>
              <button v-if="!u.deleted_at" class="tappable text-[12px] text-bad" @click="deleteUser(u)">Hapus</button>
              <button v-else class="tappable text-[12px] text-ok" @click="restoreUser(u)">Pulihkan</button>
            </div>
          </div>
        </div>
      </template>

      <!-- user form -->
      <div v-else class="mx-auto max-w-xl">
        <div class="mb-4 flex items-center gap-3">
          <button class="tappable flex h-9 w-9 items-center justify-center rounded-full bg-panel shadow-sm" @click="userForm = null" v-html="BACK_SVG"></button>
          <h2 class="m-0 text-[17px] font-bold">{{ userForm._mode === 'edit' ? 'Edit User' : 'Tambah User' }}</h2>
        </div>
        <div class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div class="inset-cell">
            <span class="label-sm w-32 shrink-0">Nama</span>
            <input v-model="userForm.name" type="text" placeholder="John Doe"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" />
          </div>
          <div v-if="userForm._mode === 'create'" class="inset-cell">
            <span class="label-sm w-32 shrink-0">Email</span>
            <input v-model="userForm.email" type="email" placeholder="john@example.com"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" />
          </div>
          <div v-if="userForm._mode === 'edit'" class="inset-cell">
            <span class="label-sm w-32 shrink-0">Status</span>
            <select v-model="userForm.status" class="bg-transparent text-right text-[14px] focus:outline-none">
              <option value="active">active</option>
              <option value="inactive">inactive</option>
            </select>
          </div>
          <div class="inset-cell">
            <span class="label-sm w-32 shrink-0">{{ userForm._mode === 'edit' ? 'Password baru' : 'Password' }}</span>
            <input v-model="userForm.password" type="password"
              :placeholder="userForm._mode === 'edit' ? 'kosongkan jika tidak diubah' : 'min 6 karakter'"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" />
          </div>
          <div class="flex gap-2 border-t px-4 py-3" :style="{ borderColor: 'var(--color-line)' }">
            <button class="btn-primary tappable flex-1" :disabled="userBusy" @click="submitUser">
              {{ userBusy ? 'Menyimpan…' : userForm._mode === 'edit' ? 'Simpan' : 'Buat User' }}
            </button>
            <button class="btn-ghost tappable" @click="userForm = null">Batal</button>
          </div>
        </div>
      </div>
    </template>

    <!-- ══ ROLES ══ -->
    <template v-if="tab === 'roles'">
      <template v-if="!roleForm && !roleDetail">
        <div class="mb-3 flex items-center justify-between">
          <span class="label-sm">{{ roles.length }} role</span>
          <button class="btn-primary tappable !px-3 !py-2 text-[13px]" @click="openRoleForm()">+ Tambah</button>
        </div>
        <div v-if="rolesLoading" class="space-y-2">
          <div v-for="i in 3" :key="i" class="h-[58px] animate-pulse rounded-[18px] bg-panel"></div>
        </div>
        <div v-else-if="!roles.length" class="rounded-[18px] bg-panel py-14 text-center shadow-sm">
          <p class="mb-0.5 text-[14px] font-semibold">Belum ada role</p>
          <p class="mb-4 text-[12px] text-mute">Role mengontrol akses user ke fitur sistem</p>
          <button class="btn-primary tappable" @click="openRoleForm()">+ Tambah Role</button>
        </div>
        <div v-else class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div v-for="r in roles" :key="r.id" class="inset-cell">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="text-[13px] font-medium">{{ r.name }}</span>
                <code class="text-[10px] text-mute">{{ r.slug }}</code>
              </div>
              <div class="text-[11px] text-mute">{{ r.description || '—' }}</div>
            </div>
            <div class="flex shrink-0 items-center gap-1.5">
              <span v-if="r.deleted_at" class="rounded-full bg-bad/10 px-1.5 py-0.5 text-[10px] font-bold text-bad">deleted</span>
              <button class="tappable rounded-lg bg-panel-2 px-2.5 py-1 text-[11px] text-mute hover:text-ink" @click="openRoleDetail(r)">Permissions</button>
              <button class="tappable text-[12px] text-accent" @click="openRoleForm(r)">Edit</button>
              <button v-if="!r.deleted_at" class="tappable text-[12px] text-bad" @click="deleteRole(r)">Hapus</button>
              <button v-else class="tappable text-[12px] text-ok" @click="restoreRole(r)">Pulihkan</button>
            </div>
          </div>
        </div>
      </template>

      <!-- role form -->
      <div v-else-if="roleForm" class="mx-auto max-w-xl">
        <div class="mb-4 flex items-center gap-3">
          <button class="tappable flex h-9 w-9 items-center justify-center rounded-full bg-panel shadow-sm" @click="roleForm = null" v-html="BACK_SVG"></button>
          <h2 class="m-0 text-[17px] font-bold">{{ roleForm.id ? 'Edit Role' : 'Tambah Role' }}</h2>
        </div>
        <div class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div class="inset-cell"><span class="label-sm w-32 shrink-0">Nama</span>
            <input v-model="roleForm.name" type="text" placeholder="Admin"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" /></div>
          <div class="inset-cell"><span class="label-sm w-32 shrink-0">Slug</span>
            <input v-model="roleForm.slug" type="text" placeholder="admin"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" /></div>
          <div class="inset-cell"><span class="label-sm w-32 shrink-0">Deskripsi</span>
            <input v-model="roleForm.description" type="text" placeholder="opsional"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" /></div>
          <div class="flex gap-2 border-t px-4 py-3" :style="{ borderColor: 'var(--color-line)' }">
            <button class="btn-primary tappable flex-1" :disabled="roleBusy" @click="submitRole">
              {{ roleBusy ? 'Menyimpan…' : roleForm.id ? 'Simpan' : 'Buat Role' }}
            </button>
            <button class="btn-ghost tappable" @click="roleForm = null">Batal</button>
          </div>
        </div>
      </div>

      <!-- permissions manager -->
      <div v-else-if="roleDetail">
        <div class="mb-4 flex items-center gap-3">
          <button class="tappable flex h-9 w-9 items-center justify-center rounded-full bg-panel shadow-sm" @click="roleDetail = null" v-html="BACK_SVG"></button>
          <div>
            <h2 class="m-0 text-[17px] font-bold">{{ roleDetail.role.name }}</h2>
            <p class="label-sm m-0">{{ roleDetail.grantedIds.size }} dari {{ allPerms.length }} permission aktif</p>
          </div>
        </div>
        <div class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div v-for="p in allPerms" :key="p.id"
            class="tappable inset-cell cursor-pointer hover:bg-panel-2"
            @click="togglePermission(p)">
            <div class="min-w-0 flex-1">
              <div class="text-[13px] font-medium">{{ p.name }}</div>
              <code class="text-[11px] text-mute">{{ p.slug }}</code>
            </div>
            <!-- toggle switch -->
            <span class="relative inline-flex h-[26px] w-[44px] shrink-0 rounded-full transition-colors"
              :class="roleDetail.grantedIds.has(p.id) ? 'bg-ok' : 'bg-panel-2'"
              style="border: 1px solid var(--color-line)">
              <span class="absolute top-[1px] h-5 w-5 rounded-full bg-white shadow transition-all"
                :class="roleDetail.grantedIds.has(p.id) ? 'left-[21px]' : 'left-[1px]'"></span>
            </span>
          </div>
          <div v-if="!allPerms.length" class="px-4 py-8 text-center text-[13px] text-mute">Belum ada permission.</div>
        </div>
      </div>
    </template>

    <!-- ══ PERMISSIONS ══ -->
    <template v-if="tab === 'permissions'">
      <template v-if="!permForm">
        <div class="mb-3 flex items-center justify-between">
          <span class="label-sm">{{ permissions.length }} permission</span>
          <button class="btn-primary tappable !px-3 !py-2 text-[13px]" @click="openPermForm()">+ Tambah</button>
        </div>
        <div v-if="permsLoading" class="space-y-2">
          <div v-for="i in 4" :key="i" class="h-12 animate-pulse rounded-[18px] bg-panel"></div>
        </div>
        <div v-else-if="!permissions.length" class="rounded-[18px] bg-panel py-14 text-center shadow-sm">
          <p class="mb-0.5 text-[14px] font-semibold">Belum ada permission</p>
          <p class="mb-4 text-[12px] text-mute">Permission mendefinisikan aksi spesifik yang bisa dilakukan</p>
          <button class="btn-primary tappable" @click="openPermForm()">+ Tambah Permission</button>
        </div>
        <div v-else class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div v-for="p in permissions" :key="p.id" class="inset-cell">
            <div class="min-w-0 flex-1">
              <div class="text-[13px] font-medium">{{ p.name }}</div>
              <div class="flex items-center gap-2 text-[11px] text-mute">
                <code>{{ p.slug }}</code>
                <span v-if="p.description" class="truncate text-mute">· {{ p.description }}</span>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <span v-if="p.deleted_at" class="rounded-full bg-bad/10 px-1.5 py-0.5 text-[10px] font-bold text-bad">deleted</span>
              <button class="tappable text-[12px] text-accent" @click="openPermForm(p)">Edit</button>
              <button v-if="!p.deleted_at" class="tappable text-[12px] text-bad" @click="deletePerm(p)">Hapus</button>
              <button v-else class="tappable text-[12px] text-ok" @click="restorePerm(p)">Pulihkan</button>
            </div>
          </div>
        </div>
      </template>

      <!-- perm form -->
      <div v-else class="mx-auto max-w-xl">
        <div class="mb-4 flex items-center gap-3">
          <button class="tappable flex h-9 w-9 items-center justify-center rounded-full bg-panel shadow-sm" @click="permForm = null" v-html="BACK_SVG"></button>
          <h2 class="m-0 text-[17px] font-bold">{{ permForm.id ? 'Edit Permission' : 'Tambah Permission' }}</h2>
        </div>
        <div class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div class="inset-cell"><span class="label-sm w-32 shrink-0">Nama</span>
            <input v-model="permForm.name" type="text" placeholder="User Read"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" /></div>
          <div class="inset-cell"><span class="label-sm w-32 shrink-0">Slug</span>
            <input v-model="permForm.slug" type="text" placeholder="user:read"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" /></div>
          <div class="inset-cell"><span class="label-sm w-32 shrink-0">Deskripsi</span>
            <input v-model="permForm.description" type="text" placeholder="opsional"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" /></div>
          <div class="flex gap-2 border-t px-4 py-3" :style="{ borderColor: 'var(--color-line)' }">
            <button class="btn-primary tappable flex-1" :disabled="permBusy" @click="submitPerm">
              {{ permBusy ? 'Menyimpan…' : permForm.id ? 'Simpan' : 'Buat Permission' }}
            </button>
            <button class="btn-ghost tappable" @click="permForm = null">Batal</button>
          </div>
        </div>
      </div>
    </template>

    <!-- ══ RATE LIMITS ══ -->
    <template v-if="tab === 'ratelimits'">
      <template v-if="!rlForm">
        <div class="mb-3 flex items-center justify-between">
          <span class="label-sm">{{ rateLimits.length }} rule</span>
          <button class="btn-primary tappable !px-3 !py-2 text-[13px]" @click="openRlForm()">+ Tambah</button>
        </div>
        <div v-if="rlLoading" class="space-y-2">
          <div v-for="i in 4" :key="i" class="h-[58px] animate-pulse rounded-[18px] bg-panel"></div>
        </div>
        <div v-else-if="!rateLimits.length" class="rounded-[18px] bg-panel py-14 text-center shadow-sm">
          <p class="mb-0.5 text-[14px] font-semibold">Belum ada rate limit rule</p>
          <p class="mb-4 text-[12px] text-mute">Rule dikache Redis, berlaku dalam 30 detik tanpa redeploy</p>
          <button class="btn-primary tappable" @click="openRlForm()">+ Tambah Rule</button>
        </div>
        <div v-else class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div v-for="r in rateLimits" :key="r.id" class="inset-cell">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="text-[13px] font-medium">{{ r.name }}</span>
                <span class="rounded-full bg-panel-2 px-1.5 py-0.5 text-[10px] font-semibold text-mute">{{ r.scope }}</span>
                <span v-if="!r.enabled" class="rounded-full bg-warn/10 px-1.5 py-0.5 text-[10px] font-semibold text-warn">off</span>
              </div>
              <div class="text-[11px] text-mute">
                {{ r.requests }}r / {{ r.window_seconds }}s
                <span v-if="r.identifier"> · <code>{{ r.identifier }}</code></span>
                <span class="ml-1 text-mute/60">· prio {{ r.priority ?? 0 }}</span>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <span v-if="r.deleted_at" class="rounded-full bg-bad/10 px-1.5 py-0.5 text-[10px] font-bold text-bad">deleted</span>
              <button class="tappable text-[12px] text-accent" @click="openRlForm(r)">Edit</button>
              <button v-if="!r.deleted_at" class="tappable text-[12px] text-bad" @click="deleteRl(r)">Hapus</button>
              <button v-else class="tappable text-[12px] text-ok" @click="restoreRl(r)">Pulihkan</button>
            </div>
          </div>
        </div>
      </template>

      <!-- rl form -->
      <div v-else class="mx-auto max-w-xl">
        <div class="mb-4 flex items-center gap-3">
          <button class="tappable flex h-9 w-9 items-center justify-center rounded-full bg-panel shadow-sm" @click="rlForm = null" v-html="BACK_SVG"></button>
          <h2 class="m-0 text-[17px] font-bold">{{ rlForm.id ? 'Edit Rule' : 'Tambah Rate Limit' }}</h2>
        </div>
        <div class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div class="inset-cell"><span class="label-sm w-36 shrink-0">Nama</span>
            <input v-model="rlForm.name" type="text" placeholder="Global default"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" /></div>
          <div class="inset-cell">
            <span class="label-sm w-36 shrink-0">Scope</span>
            <select v-model="rlForm.scope" class="bg-transparent text-right text-[14px] focus:outline-none">
              <option v-for="s in SCOPES" :key="s" :value="s">{{ s }}</option>
            </select>
          </div>
          <div class="inset-cell"><span class="label-sm w-36 shrink-0">Identifier</span>
            <input v-model="rlForm.identifier" type="text" placeholder="viewer (opsional)"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" /></div>
          <div class="inset-cell"><span class="label-sm w-36 shrink-0">Requests</span>
            <input v-model.number="rlForm.requests" type="number" min="1" max="100000"
              class="w-24 bg-transparent text-right text-[14px] focus:outline-none" /></div>
          <div class="inset-cell"><span class="label-sm w-36 shrink-0">Window (detik)</span>
            <input v-model.number="rlForm.window_seconds" type="number" min="1"
              class="w-24 bg-transparent text-right text-[14px] focus:outline-none" /></div>
          <div class="inset-cell"><span class="label-sm w-36 shrink-0">Prioritas</span>
            <input v-model.number="rlForm.priority" type="number" min="0"
              class="w-24 bg-transparent text-right text-[14px] focus:outline-none" /></div>
          <div class="px-4 py-3">
            <label class="flex items-center justify-between">
              <span class="text-[14px]">Aktif</span>
              <span class="relative inline-flex h-[30px] w-[50px] rounded-full transition-colors"
                :class="rlForm.enabled ? 'bg-accent' : 'bg-panel-2'"
                style="border:1px solid var(--color-line)" @click="rlForm.enabled = !rlForm.enabled">
                <span class="absolute top-[1px] h-6 w-6 rounded-full bg-white shadow transition-all"
                  :class="rlForm.enabled ? 'left-[23px]' : 'left-[1px]'"></span>
              </span>
            </label>
          </div>
          <div class="flex gap-2 border-t px-4 py-3" :style="{ borderColor: 'var(--color-line)' }">
            <button class="btn-primary tappable flex-1" :disabled="rlBusy" @click="submitRl">
              {{ rlBusy ? 'Menyimpan…' : rlForm.id ? 'Simpan' : 'Buat Rule' }}
            </button>
            <button class="btn-ghost tappable" @click="rlForm = null">Batal</button>
          </div>
        </div>
      </div>
    </template>

    <!-- ══ AUDIT ══ -->
    <template v-if="tab === 'audit'">
      <div class="mb-3 flex flex-col gap-2 min-[400px]:flex-row">
        <input v-model="auditSearch" type="text" placeholder="Cari aksi, resource, path…"
          class="flex-1 rounded-xl border bg-panel px-3 py-2 text-[13px] focus:outline-none"
          :style="{ borderColor: 'var(--color-line)' }" />
        <input v-model="auditResource" type="text" placeholder="resource"
          class="w-full rounded-xl border bg-panel px-3 py-2 text-[13px] focus:outline-none min-[400px]:w-32"
          :style="{ borderColor: 'var(--color-line)' }"
          @keyup.enter="loadAudit" />
        <button class="btn-ghost tappable !py-2 text-[13px]" :disabled="auditLoading" @click="loadAudit">
          {{ auditLoading ? '…' : '↺' }}
        </button>
      </div>

      <div v-if="auditLoading" class="space-y-2">
        <div v-for="i in 6" :key="i" class="h-12 animate-pulse rounded-[18px] bg-panel"></div>
      </div>

      <div v-else-if="!filteredAudit.length" class="rounded-[18px] bg-panel py-14 text-center shadow-sm">
        <p class="text-[13px] text-mute">{{ auditSearch || auditResource ? 'Tidak ada hasil' : 'Belum ada log audit.' }}</p>
      </div>

      <div v-else class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
        <template v-for="l in auditPaged" :key="l.id">
          <div class="tappable inset-cell cursor-pointer hover:bg-panel-2"
            @click="auditDetail = auditDetail?.id === l.id ? null : l">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-bold" :class="actionColor(l.action)">
                  {{ l.action }}
                </span>
                <span class="truncate text-[13px] font-medium">{{ l.resource }}</span>
                <code v-if="l.resource_id" class="shrink-0 text-[11px] text-mute">#{{ l.resource_id }}</code>
              </div>
              <div class="text-[11px] text-mute">
                {{ l.method }} {{ l.path }} · {{ fmtDate(l.created_at) }}
              </div>
            </div>
            <svg class="shrink-0 text-mute transition-transform duration-150"
              :class="auditDetail?.id === l.id ? 'rotate-90' : ''"
              width="13" height="13" viewBox="0 0 13 13" fill="none">
              <path d="M5 2.5L8.5 6.5 5 10.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>

          <!-- inline detail -->
          <Transition enter-active-class="transition duration-150 ease-out" enter-from-class="opacity-0 -translate-y-1" enter-to-class="opacity-100 translate-y-0">
            <div v-if="auditDetail?.id === l.id"
              class="border-t px-4 py-3" :style="{ borderColor: 'var(--color-line)', background: 'var(--color-panel-2)' }">
              <div class="grid grid-cols-1 gap-2 min-[400px]:grid-cols-2">
                <div class="rounded-xl bg-panel p-3">
                  <div class="label-sm mb-1.5">Sebelum</div>
                  <pre class="thin-scroll m-0 max-h-[140px] overflow-auto text-[10px] leading-relaxed text-mute">{{ prettyJson(l.old_data) }}</pre>
                </div>
                <div class="rounded-xl bg-panel p-3">
                  <div class="label-sm mb-1.5">Sesudah</div>
                  <pre class="thin-scroll m-0 max-h-[140px] overflow-auto text-[10px] leading-relaxed text-mute">{{ prettyJson(l.new_data) }}</pre>
                </div>
              </div>
              <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-mute">
                <span><span class="font-medium">User ID:</span> {{ l.user_id || '—' }}</span>
                <span><span class="font-medium">IP:</span> {{ l.ip_address || '—' }}</span>
                <span class="truncate"><span class="font-medium">Request ID:</span> {{ l.request_id || '—' }}</span>
              </div>
            </div>
          </Transition>
        </template>

        <!-- pagination -->
        <div v-if="auditPages > 1" class="flex items-center justify-between border-t px-4 py-2.5"
          :style="{ borderColor: 'var(--color-line)' }">
          <button class="tappable flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-[12px] hover:bg-panel-2 disabled:opacity-30"
            :disabled="auditPage === 1" @click="auditPage--">
            <svg width="13" height="13" viewBox="0 0 13 13" fill="none"><path d="M8 2.5L4.5 6.5l3.5 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
            Prev
          </button>
          <span class="text-[12px] text-mute">{{ auditPage }} / {{ auditPages }}</span>
          <button class="tappable flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-[12px] hover:bg-panel-2 disabled:opacity-30"
            :disabled="auditPage === auditPages" @click="auditPage++">
            Next
            <svg width="13" height="13" viewBox="0 0 13 13" fill="none"><path d="M5 2.5L8.5 6.5 5 10.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
          </button>
        </div>
      </div>
    </template>
  </div>
</template>
