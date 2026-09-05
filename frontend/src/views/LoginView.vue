<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, auth } from '../api'

const router = useRouter()
const email = ref('admin@example.com')
const password = ref('admin123')
const error = ref('')
const busy = ref(false)

async function submit() {
  busy.value = true
  error.value = ''
  try {
    const res = await api.login(email.value, password.value)
    auth.set(res.access_token, res.user)
    router.push({ name: 'dashboard' })
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="mx-auto mt-[8vh] w-full max-w-sm">
    <div class="rounded-xl border border-line bg-panel p-6">
      <h1 class="text-lg font-semibold m-0">API Gateway Console</h1>
      <p class="text-mute mt-1 mb-5">
        Sign in with a user that has the <code>admin</code> role
      </p>
      <form @submit.prevent="submit">
        <div class="mb-3.5">
          <label class="mb-1.5 block text-xs text-mute">Email</label>
          <input
            v-model.trim="email"
            type="email"
            required
            class="w-full rounded-lg border border-line bg-panel-2 px-3 py-2.5 text-ink focus:border-accent focus:outline-none"
          />
        </div>
        <div class="mb-4">
          <label class="mb-1.5 block text-xs text-mute">Password</label>
          <input
            v-model="password"
            type="password"
            required
            class="w-full rounded-lg border border-line bg-panel-2 px-3 py-2.5 text-ink focus:border-accent focus:outline-none"
          />
        </div>
        <p v-if="error" class="text-bad text-sm">{{ error }}</p>
        <button
          type="submit"
          :disabled="busy"
          class="w-full cursor-pointer rounded-lg bg-accent py-2.5 font-medium text-white transition hover:brightness-110 disabled:opacity-50"
        >
          {{ busy ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>
    </div>
    <p class="muted mt-3 text-center text-xs">
      Default bootstrap: admin@example.com / admin123
    </p>
  </div>
</template>