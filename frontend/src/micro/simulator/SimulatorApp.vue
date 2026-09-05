<script setup>
import { ref, onMounted, computed } from 'vue'
import { api } from '../../lib/api.js'
import { show } from '../../lib/toast.js'

const props = defineProps({ init: { type: String, default: '' } })

const apis = ref([])
const selectedId = ref(props.init || '')
const snippet = ref('')
const previewing = ref(false)
const publishing = ref(false)
const error = ref('')

onMounted(async () => {
  try {
    apis.value = await api.listApis()
    if (!selectedId.value) selectedId.value = String(apis.value[0]?.id || '')
    if (selectedId.value) preview()
  } catch (e) {
    error.value = e.message
  }
})

const selected = computed(() => apis.value.find((a) => a.id === Number(selectedId.value)))

async function preview() {
  if (!selectedId.value) return
  previewing.value = true
  snippet.value = ''
  try {
    snippet.value = await api.previewApi(selectedId.value)
  } catch (e) {
    show(e.message, 'err')
  } finally {
    previewing.value = false
  }
}

async function publish() {
  publishing.value = true
  try {
    await api.publish()
    show('Gateway config re-published · nginx reloaded')
    await preview()
  } catch (e) {
    show(e.message, 'err')
  } finally {
    publishing.value = false
  }
}
</script>

<template>
  <div>
    <header class="mb-4">
      <h1 class="m-0 text-[20px] font-bold tracking-tight">Configure & Simulate</h1>
      <p class="label-sm m-0 mt-0.5">Preview the exact nginx config · then publish</p>
    </header>

    <p v-if="error" class="text-bad">{{ error }}</p>

    <div class="mb-4 rounded-[18px] bg-panel p-4 shadow-sm">
      <label class="label-sm block pb-2">Registered API</label>
      <select v-model="selectedId" @change="preview"
        class="w-full rounded-xl border bg-panel-2 px-3 py-2.5 text-[14px] focus:outline-none" :style="{ borderColor: 'var(--color-line)' }">
        <option v-for="a in apis" :key="a.id" :value="a.id">{{ a.name }} — {{ a.base_path }}</option>
      </select>

      <div class="mt-3 grid grid-cols-2 gap-2 text-[12px]">
        <div class="rounded-xl bg-panel-2 px-3 py-2">
          <div class="label-sm">Upstream</div>
          <div class="truncate pt-0.5">{{ selected?.upstream }}</div>
        </div>
        <div class="rounded-xl bg-panel-2 px-3 py-2">
          <div class="label-sm">Protection</div>
          <div class="pt-0.5">{{ selected?.requires_auth ? 'JWT · auth_request' : 'open' }}</div>
        </div>
        <div class="rounded-xl bg-panel-2 px-3 py-2">
          <div class="label-sm">Methods</div>
          <div class="truncate pt-0.5">{{ selected?.methods.join(', ') }}</div>
        </div>
        <div class="rounded-xl bg-panel-2 px-3 py-2">
          <div class="label-sm">Rate / health</div>
          <div class="pt-0.5">{{ selected?.rate_limit_rpm }}r/m · {{ selected?.status }}</div>
        </div>
      </div>

      <div class="mt-3 flex items-center gap-2">
        <button class="btn-ghost tappable flex-1" :disabled="previewing" @click="preview">
          {{ previewing ? 'Rendering…' : '↻ Preview' }}
        </button>
        <button class="btn-primary tappable flex-1" :disabled="publishing" @click="publish">
          {{ publishing ? 'Publishing…' : 'Apply & reload' }}
        </button>
      </div>
    </div>

    <div class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
      <div class="flex items-center gap-2 px-4 py-3">
        <span class="h-2 w-2 rounded-full bg-ok"></span>
        <span class="text-[11px] text-mute">Generated nginx · locations/reg_{{ selectedId }}.conf</span>
      </div>
      <pre v-if="snippet" class="thin-scroll m-0 max-h-[440px] overflow-x-auto border-t px-4 py-3 text-[11px] leading-relaxed"
        :style="{ borderColor: 'var(--color-line)' }">{{ snippet }}</pre>
      <div v-else class="border-t px-4 py-10 text-center text-mute" :style="{ borderColor: 'var(--color-line)' }">
        Select an API to preview its generated config.
      </div>
    </div>
  </div>
</template>