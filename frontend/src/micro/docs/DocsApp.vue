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
      <p class="label-sm m-0 mt-0.5">APIs exposed through the gateway</p>
    </header>

    <div v-if="apis.length" class="overflow-hidden rounded-[18px] bg-panel shadow-sm">
      <div v-for="a in apis" :key="a.id" class="inset-cell">
          <div class="min-w-0">
            <code class="text-accent">{{ a.base_path }}/*</code>
            <div class="truncate text-[11px] text-mute">{{ a.methods.join(', ') }} → {{ a.upstream }}</div>
          </div>
          <span class="shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold" :class="a.requires_auth ? 'bg-accent-tint text-accent-2' : 'bg-panel-2 text-mute'">
            {{ a.requires_auth ? 'JWT' : 'open' }}
          </span>
          <a href="/admin/simulate" class="shrink-0 text-[12px] font-medium text-accent-2">Test →</a>
        </div>
      </div>
  </div>
</template>