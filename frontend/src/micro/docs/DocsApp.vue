<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../../lib/api.js'

const apis = ref([])
onMounted(async () => {
  try {
    apis.value = await api.listApis()
  } catch {
    /* docs stays usable without list */
  }
})
</script>

<template>
  <div>
    <header class="mb-4">
      <h1 class="m-0 text-[20px] font-bold tracking-tight">Documentation</h1>
      <p class="label-sm m-0 mt-0.5">OpenAPI (Swagger UI) + dynamically exposed routes</p>
    </header>

    <div class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
      <div class="flex h-[58vh] items-center justify-center">
        <iframe src="/swagger/index.html" class="h-full w-full border-0" title="Swagger UI"></iframe>
      </div>
    </div>

    <div v-if="apis.length" class="mt-4 overflow-hidden rounded-[18px] bg-panel shadow-sm">
      <div class="label-sm border-b px-4 py-2.5" :style="{ borderColor: 'var(--color-line)' }">
        Dynamically exposed routes
      </div>
      <div class="divide-y divide-solid" :style="{ borderColor: 'var(--color-line)' }">
        <div v-for="a in apis" :key="a.id" class="inset-cell">
          <div class="min-w-0">
            <code class="text-accent">{{ a.base_path }}/*</code>
            <div class="truncate text-[11px] text-mute">{{ a.methods.join(', ') }} → {{ a.upstream }}</div>
          </div>
          <span class="shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold" :class="a.requires_auth ? 'bg-accent-tint text-accent-2' : 'bg-panel-2 text-mute'">
            {{ a.requires_auth ? 'JWT' : 'open' }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>