<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { api, auth } from '../lib/api.js'

const router = useRouter()
const route = useRoute()
const email = ref('admin@example.com')
const password = ref('admin123')
const error = ref('')
const busy = ref(false)

async function submit() {
  busy.value = true
  error.value = ''
  try {
    const s = await api.login(email.value, password.value)
    auth.set(s.access_token, s.user)
    router.push(String(route.query.next || '/'))
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-bg px-5">
    <div class="w-full max-w-sm">
      <div class="mb-6 flex flex-col items-center text-center">
        <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-accent shadow-lg shadow-accent/20">
          <svg width="26" height="26" viewBox="0 0 24 24" class="text-white"><path fill="none" stroke="currentColor" stroke-width="2" d="M4 6h16v4H4zM4 12h16v4H4zM4 18h10"/></svg>
        </div>
        <h1 class="mt-4 mb-1 text-[22px] font-bold tracking-tight">Gateway Console</h1>
        <p class="m-0 text-[13px] text-mute">go-enterprise-api · API Gateway manager</p>
      </div>

      <form @submit.prevent="submit" class="overflow-hidden rounded-[22px] bg-panel shadow-xl shadow-black/5">
        <div class="inset-cell">
          <span class="label-sm w-20 shrink-0">Email</span>
          <input v-model.trim="email" type="email" required autocomplete="username"
            class="w-full bg-transparent px-0 text-right text-[14px] focus:outline-none" />
        </div>
        <div class="inset-cell">
          <span class="label-sm w-20 shrink-0">Password</span>
          <input v-model="password" type="password" required autocomplete="current-password"
            class="w-full bg-transparent px-0 text-right text-[14px] focus:outline-none" />
        </div>
        <p v-if="error" class="m-0 bg-bad/5 px-4 py-2 text-[12px] text-bad">{{ error }}</p>
        <div class="p-4">
          <button type="submit" class="btn-primary tappable w-full py-2.5" :disabled="busy">
            {{ busy ? 'Signing in…' : 'Sign in' }}
          </button>
        </div>
      </form>

      <p class="mt-5 text-center text-[11px] text-mute">Default demo admin: admin@example.com / admin123</p>
    </div>
  </div>
</template>