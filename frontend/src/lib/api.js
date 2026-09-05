const TOKEN_KEY = 'gw_admin_token'
const USER_KEY = 'gw_admin_user'

export const auth = {
  get token() {
    return localStorage.getItem(TOKEN_KEY) || ''
  },
  get user() {
    const raw = localStorage.getItem(USER_KEY)
    return raw ? JSON.parse(raw) : null
  },
  set(token, user) {
    localStorage.setItem(TOKEN_KEY, token)
    localStorage.setItem(USER_KEY, JSON.stringify(user))
  },
  clear() {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  }
}

async function request(method, path, body) {
  const headers = { Accept: 'application/json' }
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (auth.token) headers['Authorization'] = 'Bearer ' + auth.token

  const res = await fetch('/api/v1' + path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined
  })

  if (res.status === 401 && auth.token) {
    auth.clear()
    window.location.href = '/admin/login'
    throw new Error('session expired')
  }

  if (res.status === 204) return null

  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const err = new Error(data?.error?.message || res.statusText)
    err.status = res.status
    throw err
  }
  return data?.data ?? data
}

export const api = {
  login: (email, password) =>
    request('POST', '/auth/login', { email, password }),
  logout: () => request('POST', '/auth/logout'),
  me: () => request('GET', '/auth/me'),

  listApis: () => request('GET', '/gateway/apis'),
  getApi: (id) => request('GET', `/gateway/apis/${id}`),
  createApi: (payload) => request('POST', '/gateway/apis', payload),
  updateApi: (id, payload) => request('PUT', `/gateway/apis/${id}`, payload),
  deleteApi: (id) => request('DELETE', `/gateway/apis/${id}`),
  restoreApi: (id) => request('POST', `/gateway/apis/${id}/restore`),
  previewApi: (id) => request('GET', `/gateway/apis/${id}/preview`),
  statsApi: (id) => request('GET', `/gateway/apis/${id}/stats`),
  publish: () => request('POST', '/gateway/publish'),

  // ── users ──
  listUsers: () => request('GET', '/users'),
  getUser: (id) => request('GET', `/users/${id}`),
  createUser: (p) => request('POST', '/users', p),
  updateUser: (id, p) => request('PUT', `/users/${id}`, p),
  deleteUser: (id) => request('DELETE', `/users/${id}`),
  restoreUser: (id) => request('POST', `/users/${id}/restore`),

  // ── roles ──
  listRoles: () => request('GET', '/roles'),
  getRole: (id) => request('GET', `/roles/${id}`),
  createRole: (p) => request('POST', '/roles', p),
  updateRole: (id, p) => request('PUT', `/roles/${id}`, p),
  deleteRole: (id) => request('DELETE', `/roles/${id}`),
  restoreRole: (id) => request('POST', `/roles/${id}/restore`),
  grantPermission: (roleId, permissionId) => request('POST', `/roles/${roleId}/permissions`, { permission_id: permissionId }),
  revokePermission: (roleId, permissionId) => request('DELETE', `/roles/${roleId}/permissions/${permissionId}`),

  // ── permissions ──
  listPermissions: () => request('GET', '/permissions'),
  createPermission: (p) => request('POST', '/permissions', p),
  updatePermission: (id, p) => request('PUT', `/permissions/${id}`, p),
  deletePermission: (id) => request('DELETE', `/permissions/${id}`),
  restorePermission: (id) => request('POST', `/permissions/${id}/restore`),

  // ── rate limits ──
  listRateLimits: () => request('GET', '/rate-limits'),
  createRateLimit: (p) => request('POST', '/rate-limits', p),
  updateRateLimit: (id, p) => request('PUT', `/rate-limits/${id}`, p),
  deleteRateLimit: (id) => request('DELETE', `/rate-limits/${id}`),
  restoreRateLimit: (id) => request('POST', `/rate-limits/${id}/restore`),

  // ── audit logs ──
  listAuditLogs: (params) => {
    const q = params ? '?' + new URLSearchParams(params).toString() : ''
    return request('GET', `/audit-logs${q}`)
  },
}