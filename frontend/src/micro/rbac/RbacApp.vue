<script setup>
import { ref, computed, onMounted, watch } from 'vue'
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

// ── shared helpers ──
function fmtDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleString('id-ID', { dateStyle: 'short', timeStyle: 'short' })
}

// ═══════════════════════════════════════════════
// USERS
// ═══════════════════════════════════════════════
const users = ref([])
const usersLoading = ref(false)
const userForm = ref(null) // null = list, {} = form
const userBusy = ref(false)
const userSearch = ref('')

const filteredUsers = computed(() => {
  const q = userSearch.value.toLowerCase()
  return users.value.filter(u =>
    !q || u.email?.toLowerCase().includes(q) || u.display_name?.toLowerCase().includes(q)
  )
})

async function loadUsers() {
  usersLoading.value = true
  try { users.value = await api.listUsers() } catch (e) { show(e.message) } finally { usersLoading.value = false }
}

function openUserForm(u = null) {
  userForm.value = u
    ? { id: u.id, email: u.email, display_name: u.display_name, password: '' }
    : { id: null, email: '', display_name: '', password: '' }
}

async function submitUser() {
  userBusy.value = true
  try {
    const { id, password, ...rest } = userForm.value
    const payload = { ...rest, ...(password ? { password } : {}) }
    if (id) { await api.updateUser(id, payload); show('User diperbarui') }
    else { await api.createUser({ ...payload, password }); show('User dibuat') }
    userForm.value = null
    loadUsers()
  } catch (e) { show(e.message) } finally { userBusy.value = false }
}

async function deleteUser(u) {
  if (!confirm(`Hapus ${u.email}?`)) return
  try { await api.deleteUser(u.id); show('User dihapus'); loadUsers() } catch (e) { show(e.message) }
}

async function restoreUser(u) {
  try { await api.restoreUser(u.id); show('User dipulihkan'); loadUsers() } catch (e) { show(e.message) }
}

// ═══════════════════════════════════════════════
// ROLES
// ═══════════════════════════════════════════════
const roles = ref([])
const permissions = ref([])
const rolesLoading = ref(false)
const roleForm = ref(null)
const roleBusy = ref(false)
const roleDetail = ref(null) // role yang sedang dikelola permissions-nya

async function loadRoles() {
  rolesLoading.value = true
  try {
    [roles.value, permissions.value] = await Promise.all([api.listRoles(), api.listPermissions()])
  } catch (e) { show(e.message) } finally { rolesLoading.value = false }
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
  } catch (e) { show(e.message) } finally { roleBusy.value = false }
}

async function deleteRole(r) {
  if (!confirm(`Hapus role ${r.name}?`)) return
  try { await api.deleteRole(r.id); show('Role dihapus'); loadRoles() } catch (e) { show(e.message) }
}

async function restoreRole(r) {
  try { await api.restoreRole(r.id); show('Role dipulihkan'); loadRoles() } catch (e) { show(e.message) }
}

function hasPermission(role, permId) {
  return role.permissions?.some(p => p.id === permId)
}

async function togglePermission(role, perm) {
  try {
    if (hasPermission(role, perm.id)) {
      await api.revokePermission(role.id, perm.id)
      show(`${perm.slug} dicabut`)
    } else {
      await api.grantPermission(role.id, perm.id)
      show(`${perm.slug} diberikan`)
    }
    loadRoles()
  } catch (e) { show(e.message) }
}

// ═══════════════════════════════════════════════
// PERMISSIONS
// ═══════════════════════════════════════════════
const permsLoading = ref(false)
const permForm = ref(null)
const permBusy = ref(false)

async function loadPermissions() {
  permsLoading.value = true
  try { permissions.value = await api.listPermissions() } catch (e) { show(e.message) } finally { permsLoading.value = false }
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
  } catch (e) { show(e.message) } finally { permBusy.value = false }
}

async function deletePerm(p) {
  if (!confirm(`Hapus permission ${p.slug}?`)) return
  try { await api.deletePermission(p.id); show('Permission dihapus'); loadPermissions() } catch (e) { show(e.message) }
}

async function restorePerm(p) {
  try { await api.restorePermission(p.id); show('Permission dipulihkan'); loadPermissions() } catch (e) { show(e.message) }
}

// ═══════════════════════════════════════════════
// RATE LIMITS
// ═══════════════════════════════════════════════
const rateLimits = ref([])
const rlLoading = ref(false)
const rlForm = ref(null)
const rlBusy = ref(false)
const SCOPES = ['global', 'ip', 'user', 'role', 'route', 'api_key']

async function loadRateLimits() {
  rlLoading.value = true
  try { rateLimits.value = await api.listRateLimits() } catch (e) { show(e.message) } finally { rlLoading.value = false }
}

function openRlForm(r = null) {
  rlForm.value = r
    ? { id: r.id, name: r.name, scope: r.scope, identifier: r.identifier, requests: r.requests, window_seconds: r.window_seconds }
    : { id: null, name: '', scope: 'ip', identifier: '', requests: 60, window_seconds: 60 }
}

async function submitRl() {
  rlBusy.value = true
  try {
    const { id, ...rest } = rlForm.value
    if (id) { await api.updateRateLimit(id, rest); show('Rate limit diperbarui') }
    else { await api.createRateLimit(rest); show('Rate limit dibuat') }
    rlForm.value = null
    loadRateLimits()
  } catch (e) { show(e.message) } finally { rlBusy.value = false }
}

async function deleteRl(r) {
  if (!confirm(`Hapus rule ${r.name}?`)) return
  try { await api.deleteRateLimit(r.id); show('Rate limit dihapus'); loadRateLimits() } catch (e) { show(e.message) }
}

async function restoreRl(r) {
  try { await api.restoreRateLimit(r.id); show('Rate limit dipulihkan'); loadRateLimits() } catch (e) { show(e.message) }
}

// ═══════════════════════════════════════════════
// AUDIT LOGS
// ═══════════════════════════════════════════════
const auditLogs = ref([])
const auditLoading = ref(false)
const auditSearch = ref('')
const auditResource = ref('')
const auditPage = ref(1)
const AUDIT_PER_PAGE = 15

const filteredAudit = computed(() => {
  const q = auditSearch.value.toLowerCase()
  return auditLogs.value.filter(l =>
    !q || l.action?.toLowerCase().includes(q) || l.resource?.toLowerCase().includes(q) || l.actor_email?.toLowerCase().includes(q)
  )
})

const auditPages = computed(() => Math.max(1, Math.ceil(filteredAudit.value.length / AUDIT_PER_PAGE)))
const auditPaged = computed(() => {
  const s = (auditPage.value - 1) * AUDIT_PER_PAGE
  return filteredAudit.value.slice(s, s + AUDIT_PER_PAGE)
})
watch(filteredAudit, () => { auditPage.value = 1 })

async function loadAudit() {
  auditLoading.value = true
  try {
    const params = {}
    if (auditResource.value) params.resource = auditResource.value
    auditLogs.value = await api.listAuditLogs(params)
  } catch (e) { show(e.message) } finally { auditLoading.value = false }
}

const auditDetail = ref(null)

// ── load on tab switch ──
watch(tab, (t) => {
  if (t === 'users' && !users.value.length) loadUsers()
  if (t === 'roles' && !roles.value.length) loadRoles()
  if (t === 'permissions' && !permissions.value.length) loadPermissions()
  if (t === 'ratelimits' && !rateLimits.value.length) loadRateLimits()
  if (t === 'audit' && !auditLogs.value.length) loadAudit()
}, { immediate: true })
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

    <!-- ══════════════ USERS ══════════════ -->
    <div v-if="tab === 'users'">
      <!-- list -->
      <template v-if="!userForm">
        <div class="mb-3 flex items-center gap-2">
          <input v-model="userSearch" type="text" placeholder="Cari email atau nama…"
            class="flex-1 rounded-xl border bg-panel px-3 py-2 text-[13px] focus:outline-none"
            :style="{ borderColor: 'var(--color-line)' }" />
          <button class="btn-primary tappable !px-3 !py-2 text-[13px]" @click="openUserForm()">+ Tambah</button>
        </div>

        <div v-if="usersLoading" class="space-y-2">
          <div v-for="i in 4" :key="i" class="h-14 animate-pulse rounded-[18px] bg-panel"></div>
        </div>

        <div v-else-if="!filteredUsers.length" class="rounded-[18px] bg-panel py-12 text-center shadow-sm">
          <p class="text-[13px] text-mute">Belum ada user.</p>
          <button class="btn-primary tappable mt-3" @click="openUserForm()">+ Tambah User</button>
        </div>

        <div v-else class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div v-for="u in filteredUsers" :key="u.id"
            class="inset-cell">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <div class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent-tint)] text-[11px] font-bold text-accent">
                  {{ (u.display_name || u.email || '?')[0].toUpperCase() }}
                </div>
                <div class="min-w-0">
                  <div class="truncate text-[13px] font-medium">{{ u.display_name || '—' }}</div>
                  <div class="truncate text-[11px] text-mute">{{ u.email }}</div>
                </div>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <span v-if="u.deleted_at" class="rounded-full bg-bad/10 px-2 py-0.5 text-[10px] font-bold text-bad">deleted</span>
              <button class="tappable text-[12px] text-accent" @click="openUserForm(u)">Edit</button>
              <button v-if="!u.deleted_at" class="tappable text-[12px] text-bad" @click="deleteUser(u)">Hapus</button>
              <button v-else class="tappable text-[12px] text-ok" @click="restoreUser(u)">Pulihkan</button>
            </div>
          </div>
        </div>
      </template>

      <!-- form -->
      <div v-else class="mx-auto max-w-xl">
        <div class="mb-4 flex items-center gap-3">
          <button class="tappable flex h-9 w-9 items-center justify-center rounded-full bg-panel shadow-sm" @click="userForm = null">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none"><path d="M15 6l-6 6 6 6" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/></svg>
          </button>
          <h2 class="m-0 text-[17px] font-bold">{{ userForm.id ? 'Edit User' : 'Tambah User' }}</h2>
        </div>
        <div class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div class="inset-cell"><span class="label-sm w-32 shrink-0">Nama</span>
            <input v-model="userForm.display_name" type="text" placeholder="John Doe"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" /></div>
          <div class="inset-cell"><span class="label-sm w-32 shrink-0">Email</span>
            <input v-model="userForm.email" type="email" placeholder="john@example.com"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" /></div>
          <div class="inset-cell"><span class="label-sm w-32 shrink-0">{{ userForm.id ? 'Password baru' : 'Password' }}</span>
            <input v-model="userForm.password" type="password" :placeholder="userForm.id ? 'kosongkan jika tidak diubah' : 'min 6 karakter'"
              class="w-full bg-transparent text-right text-[14px] focus:outline-none placeholder:text-mute/50" /></div>
          <div class="flex gap-2 border-t px-4 py-3" :style="{ borderColor: 'var(--color-line)' }">
            <button class="btn-primary tappable flex-1" :disabled="userBusy" @click="submitUser">
              {{ userBusy ? 'Menyimpan…' : userForm.id ? 'Simpan' : 'Buat User' }}
            </button>
            <button class="btn-ghost tappable" @click="userForm = null">Batal</button>
          </div>
        </div>
      </div>
    </div>

    <!-- ══════════════ ROLES ══════════════ -->
    <div v-if="tab === 'roles'">
      <template v-if="!roleForm && !roleDetail">
        <div class="mb-3 flex items-center justify-between">
          <span class="label-sm">{{ roles.length }} role</span>
          <button class="btn-primary tappable !px-3 !py-2 text-[13px]" @click="openRoleForm()">+ Tambah</button>
        </div>
        <div v-if="rolesLoading" class="space-y-2">
          <div v-for="i in 3" :key="i" class="h-14 animate-pulse rounded-[18px] bg-panel"></div>
        </div>
        <div v-else-if="!roles.length" class="rounded-[18px] bg-panel py-12 text-center shadow-sm">
          <p class="text-[13px] text-mute">Belum ada role.</p>
          <button class="btn-primary tappable mt-3" @click="openRoleForm()">+ Tambah Role</button>
        </div>
        <div v-else class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div v-for="r in roles" :key="r.id" class="inset-cell">
            <div class="min-w-0 flex-1">
              <div class="text-[13px] font-medium">{{ r.name }}</div>
              <div class="flex items-center gap-2 text-[11px] text-mute">
                <code>{{ r.slug }}</code>
                <span>· {{ r.permissions?.length ?? 0 }} permission</span>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <span v-if="r.deleted_at" class="rounded-full bg-bad/10 px-2 py-0.5 text-[10px] font-bold text-bad">deleted</span>
              <button class="tappable text-[12px] text-mute" @click="roleDetail = r; loadRoles()">Permissions</button>
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
          <button class="tappable flex h-9 w-9 items-center justify-center rounded-full bg-panel shadow-sm" @click="roleForm = null">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none"><path d="M15 6l-6 6 6 6" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/></svg>
          </button>
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

      <!-- role permissions manager -->
      <div v-else-if="roleDetail">
        <div class="mb-4 flex items-center gap-3">
          <button class="tappable flex h-9 w-9 items-center justify-center rounded-full bg-panel shadow-sm" @click="roleDetail = null">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none"><path d="M15 6l-6 6 6 6" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/></svg>
          </button>
          <div>
            <h2 class="m-0 text-[17px] font-bold">{{ roleDetail.name }}</h2>
            <p class="label-sm m-0">Kelola permissions</p>
          </div>
        </div>
        <div class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div v-for="p in permissions" :key="p.id" class="inset-cell">
            <div class="min-w-0 flex-1">
              <div class="text-[13px] font-medium">{{ p.name }}</div>
              <code class="text-[11px] text-mute">{{ p.slug }}</code>
            </div>
            <button
              class="tappable shrink-0 rounded-xl px-3 py-1.5 text-[12px] font-medium transition-colors"
              :class="hasPermission(roleDetail, p.id)
                ? 'bg-ok/10 text-ok'
                : 'bg-panel-2 text-mute hover:text-ink'"
              @click="togglePermission(roleDetail, p)">
              {{ hasPermission(roleDetail, p.id) ? '✓ Diberikan' : 'Berikan' }}
            </button>
          </div>
          <div v-if="!permissions.length" class="px-4 py-8 text-center text-[13px] text-mute">Belum ada permission.</div>
        </div>
      </div>
    </div>

    <!-- ══════════════ PERMISSIONS ══════════════ -->
    <div v-if="tab === 'permissions'">
      <template v-if="!permForm">
        <div class="mb-3 flex items-center justify-between">
          <span class="label-sm">{{ permissions.length }} permission</span>
          <button class="btn-primary tappable !px-3 !py-2 text-[13px]" @click="openPermForm()">+ Tambah</button>
        </div>
        <div v-if="permsLoading" class="space-y-2">
          <div v-for="i in 4" :key="i" class="h-12 animate-pulse rounded-[18px] bg-panel"></div>
        </div>
        <div v-else-if="!permissions.length" class="rounded-[18px] bg-panel py-12 text-center shadow-sm">
          <p class="text-[13px] text-mute">Belum ada permission.</p>
          <button class="btn-primary tappable mt-3" @click="openPermForm()">+ Tambah Permission</button>
        </div>
        <div v-else class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div v-for="p in permissions" :key="p.id" class="inset-cell">
            <div class="min-w-0 flex-1">
              <div class="text-[13px] font-medium">{{ p.name }}</div>
              <div class="flex items-center gap-2 text-[11px] text-mute">
                <code>{{ p.slug }}</code>
                <span v-if="p.description" class="truncate">· {{ p.description }}</span>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <span v-if="p.deleted_at" class="rounded-full bg-bad/10 px-2 py-0.5 text-[10px] font-bold text-bad">deleted</span>
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
          <button class="tappable flex h-9 w-9 items-center justify-center rounded-full bg-panel shadow-sm" @click="permForm = null">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none"><path d="M15 6l-6 6 6 6" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/></svg>
          </button>
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
    </div>

    <!-- ══════════════ RATE LIMITS ══════════════ -->
    <div v-if="tab === 'ratelimits'">
      <template v-if="!rlForm">
        <div class="mb-3 flex items-center justify-between">
          <span class="label-sm">{{ rateLimits.length }} rule</span>
          <button class="btn-primary tappable !px-3 !py-2 text-[13px]" @click="openRlForm()">+ Tambah</button>
        </div>
        <div v-if="rlLoading" class="space-y-2">
          <div v-for="i in 4" :key="i" class="h-14 animate-pulse rounded-[18px] bg-panel"></div>
        </div>
        <div v-else-if="!rateLimits.length" class="rounded-[18px] bg-panel py-12 text-center shadow-sm">
          <p class="text-[13px] text-mute">Belum ada rate limit rule.</p>
          <button class="btn-primary tappable mt-3" @click="openRlForm()">+ Tambah Rule</button>
        </div>
        <div v-else class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
          <div v-for="r in rateLimits" :key="r.id" class="inset-cell">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="text-[13px] font-medium">{{ r.name }}</span>
                <span class="rounded-full bg-panel-2 px-1.5 py-0.5 text-[10px] font-bold text-mute">{{ r.scope }}</span>
              </div>
              <div class="text-[11px] text-mute">
                {{ r.requests }}r / {{ r.window_seconds }}s
                <span v-if="r.identifier"> · <code>{{ r.identifier }}</code></span>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <span v-if="r.deleted_at" class="rounded-full bg-bad/10 px-2 py-0.5 text-[10px] font-bold text-bad">deleted</span>
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
          <button class="tappable flex h-9 w-9 items-center justify-center rounded-full bg-panel shadow-sm" @click="rlForm = null">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none"><path d="M15 6l-6 6 6 6" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/></svg>
          </button>
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
            <input v-model.number="rlForm.requests" type="number" min="1"
              class="w-24 bg-transparent text-right text-[14px] focus:outline-none" /></div>
          <div class="inset-cell"><span class="label-sm w-36 shrink-0">Window (detik)</span>
            <input v-model.number="rlForm.window_seconds" type="number" min="1"
              class="w-24 bg-transparent text-right text-[14px] focus:outline-none" /></div>
          <div class="flex gap-2 border-t px-4 py-3" :style="{ borderColor: 'var(--color-line)' }">
            <button class="btn-primary tappable flex-1" :disabled="rlBusy" @click="submitRl">
              {{ rlBusy ? 'Menyimpan…' : rlForm.id ? 'Simpan' : 'Buat Rule' }}
            </button>
            <button class="btn-ghost tappable" @click="rlForm = null">Batal</button>
          </div>
        </div>
      </div>
    </div>

    <!-- ══════════════ AUDIT LOGS ══════════════ -->
    <div v-if="tab === 'audit'">
      <!-- filter bar -->
      <div class="mb-3 flex flex-col gap-2 min-[400px]:flex-row">
        <input v-model="auditSearch" type="text" placeholder="Cari aksi, resource, email…"
          class="flex-1 rounded-xl border bg-panel px-3 py-2 text-[13px] focus:outline-none"
          :style="{ borderColor: 'var(--color-line)' }" />
        <input v-model="auditResource" type="text" placeholder="Filter resource"
          class="w-full rounded-xl border bg-panel px-3 py-2 text-[13px] focus:outline-none min-[400px]:w-36"
          :style="{ borderColor: 'var(--color-line)' }"
          @keyup.enter="loadAudit" />
        <button class="btn-ghost tappable !py-2 text-[13px]" @click="loadAudit">↺ Muat</button>
      </div>

      <div v-if="auditLoading" class="space-y-2">
        <div v-for="i in 6" :key="i" class="h-12 animate-pulse rounded-[18px] bg-panel"></div>
      </div>

      <div v-else-if="!filteredAudit.length" class="rounded-[18px] bg-panel py-12 text-center shadow-sm">
        <p class="text-[13px] text-mute">Belum ada log audit.</p>
      </div>

      <div v-else class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
        <!-- rows -->
        <div v-for="l in auditPaged" :key="l.id"
          class="tappable inset-cell cursor-pointer hover:bg-panel-2"
          @click="auditDetail = auditDetail?.id === l.id ? null : l">
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-bold"
                :class="{
                  'bg-ok/10 text-ok': ['create','restore'].includes(l.action),
                  'bg-bad/10 text-bad': l.action === 'delete',
                  'bg-warn/10 text-warn': l.action === 'update',
                  'bg-panel-2 text-mute': !['create','delete','update','restore'].includes(l.action)
                }">{{ l.action }}</span>
              <span class="truncate text-[13px] font-medium">{{ l.resource }}</span>
              <code v-if="l.resource_id" class="shrink-0 text-[11px] text-mute">#{{ l.resource_id }}</code>
            </div>
            <div class="text-[11px] text-mute">{{ l.actor_email || '—' }} · {{ fmtDate(l.created_at) }}</div>
          </div>
          <svg class="shrink-0 text-mute transition-transform" :class="{ 'rotate-90': auditDetail?.id === l.id }"
            width="14" height="14" viewBox="0 0 14 14" fill="none">
            <path d="M5 3l4 4-4 4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </div>

        <!-- detail expand -->
        <div v-if="auditDetail" class="border-t px-4 py-3" :style="{ borderColor: 'var(--color-line)' }">
          <div class="label-sm mb-2">Detail perubahan</div>
          <div class="grid grid-cols-1 gap-2 min-[400px]:grid-cols-2">
            <div class="rounded-xl bg-panel-2 p-3">
              <div class="label-sm mb-1">Sebelum</div>
              <pre class="thin-scroll m-0 max-h-[160px] overflow-auto text-[10px] leading-relaxed">{{ auditDetail.old_data ? JSON.stringify(JSON.parse(auditDetail.old_data), null, 2) : '—' }}</pre>
            </div>
            <div class="rounded-xl bg-panel-2 p-3">
              <div class="label-sm mb-1">Sesudah</div>
              <pre class="thin-scroll m-0 max-h-[160px] overflow-auto text-[10px] leading-relaxed">{{ auditDetail.new_data ? JSON.stringify(JSON.parse(auditDetail.new_data), null, 2) : '—' }}</pre>
            </div>
          </div>
          <div class="mt-2 grid grid-cols-2 gap-2 text-[11px]">
            <div><span class="text-mute">IP: </span>{{ auditDetail.ip_address || '—' }}</div>
            <div><span class="text-mute">UA: </span><span class="truncate">{{ auditDetail.user_agent || '—' }}</span></div>
          </div>
        </div>

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
    </div>
  </div>
</template>
