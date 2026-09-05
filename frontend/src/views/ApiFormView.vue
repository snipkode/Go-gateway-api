<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { show } from '../toast'

const route = useRoute()
const router = useRouter()
const editing = computed(() => !!route.params.id)

const form = ref({
  name: '',
  base_path: '',
  upstream: '',
  methods: ['GET'],
  requires_auth: true,
  rate_limit_rpm: 60,
  is_active: true,
  note: ''
})
const allMethods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']
const error = ref('')
const busy = ref(false)

onMounted(async () => {
  if (!editing.value) return
  try {
    const a = await api.getApi(route.params.id)
    form.value = {
      name: a.name,
      base_path: a.base_path,
      upstream: a.upstream,
      methods: a.methods,
      requires_auth: a.requires_auth,
      rate_limit_rpm: a.rate_limit_rpm,
      is_active: a.is_active,
      note: a.note || ''
    }
  } catch (e) {
    error.value = e.message
  }
})

function toggleMethod(m) {
  const i = form.value.methods.indexOf(m)
  if (i >= 0) form.value.methods.splice(i, 1)
  else form.value.methods.push(m)
}

async function submit() {
  busy.value = true
  error.value = ''
  try {
    const payload = { ...form.value, methods: form.value.methods.length ? form.value.methods : ['GET'] }
    if (editing.value) {
      await api.updateApi(route.params.id, payload)
      show('Registered API updated · gateway reloaded')
    } else {
      const created = await api.createApi(payload)
      show(`"${created.base_path}" registered · now live on the gateway`)
    }
    router.push({ name: 'apis' })
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="max-w-3xl">
    <h1 class="m-0 text-xl font-semibold">{{ editing ? 'Edit registered API' : 'Register an API' }}</h1>
    <p class="text-mute mt-1 mb-5">
      The gateway exposes it at <code>{{ form.base_path || '/<your-path>' }}/*</code> → upstream,
      with a hot-reload of the nginx config on save.
    </p>

    <div class="rounded-xl border border-line bg-panel p-6">
      <p v-if="error" class="text-bad">{{ error }}</p>
      <form @submit.prevent="submit">
        <div class="mb-4">
          <label class="mb-1.5 block text-xs text-mute">Name</label>
          <input v-model.trim="form.name" type="text" required placeholder="Order Service"
            class="w-full rounded-lg border border-line bg-panel-2 px-3 py-2.5 focus:border-accent focus:outline-none" />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div class="mb-4">
            <label class="mb-1.5 block text-xs text-mute">Base path (single segment)</label>
            <input v-model.trim="form.base_path" type="text" required placeholder="/orders"
              pattern="/[a-zA-Z0-9._~-]+"
              class="w-full rounded-lg border border-line bg-panel-2 px-3 py-2.5 focus:border-accent focus:outline-none" />
          </div>
          <div class="mb-4">
            <label class="mb-1.5 block text-xs text-mute">Upstream origin</label>
            <input v-model.trim="form.upstream" type="text" required placeholder="http://orders-service:8080"
              class="w-full rounded-lg border border-line bg-panel-2 px-3 py-2.5 focus:border-accent focus:outline-none" />
          </div>
        </div>
        <div class="mb-4">
          <label class="mb-1.5 block text-xs text-mute">Allowed methods</label>
          <div class="flex flex-wrap gap-2">
            <button v-for="m in allMethods" type="button" :key="m"
              class="rounded-md px-2.5 py-1 text-xs transition"
              :class="form.methods.includes(m) ? 'bg-accent text-white' : 'bg-panel-2 text-mute border border-line'"
              @click="toggleMethod(m)">
              {{ m }}
            </button>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div class="mb-4">
            <label class="mb-1.5 block text-xs text-mute">Rate limit (req/min, per IP)</label>
            <input v-model.number="form.rate_limit_rpm" type="number" min="1" max="100000" required
              class="w-full rounded-lg border border-line bg-panel-2 px-3 py-2.5 focus:border-accent focus:outline-none" />
          </div>
          <div class="mb-4">
            <label class="mb-1.5 block text-xs text-mute">Note (optional)</label>
            <input v-model.trim="form.note" type="text" placeholder="owner, docs link, …"
              class="w-full rounded-lg border border-line bg-panel-2 px-3 py-2.5 focus:border-accent focus:outline-none" />
          </div>
        </div>

        <div class="mb-6 flex gap-6">
          <label class="flex items-center gap-2 text-sm">
            <input v-model="form.requires_auth" type="checkbox" class="accent-accent" />
            Require JWT (auth_request)
          </label>
          <label class="flex items-center gap-2 text-sm">
            <input v-model="form.is_active" type="checkbox" class="accent-accent" />
            Active (published to gateway)
          </label>
        </div>

        <div class="flex gap-3">
          <button type="submit" :disabled="busy" class="rounded-lg bg-accent px-5 py-2.5 font-medium text-white hover:brightness-110 disabled:opacity-50">
            {{ busy ? 'Saving…' : editing ? 'Save & re-publish' : 'Register & expose' }}
          </button>
          <button type="button" class="rounded-lg border border-line bg-panel-2 px-5 py-2.5 hover:brightness-115" @click="router.push({ name: 'apis' })">
            Cancel
          </button>
        </div>
      </form>
    </div>
  </div>
</template>