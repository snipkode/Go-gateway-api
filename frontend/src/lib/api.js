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
  publish: () => request('POST', '/gateway/publish')
}