<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'

const apis = ref([])
const swaggerUrl = '/swagger/index.html'

onMounted(async () => {
  try {
    apis.value = await api.listApis()
  } catch {
    /* docs tab stays usable without auth introspection */
  }
})
</script>

<template>
  <div>
    <h1 class="m-0 text-xl font-semibold">Gateway Documentation</h1>
    <p class="text-mute mt-1 mb-5">
      OpenAPI spec of the Go API (Swagger UI) plus every dynamically registered route.
    </p>

    <div class="grid grid-cols-3 gap-4">
      <div class="col-span-2 overflow-hidden rounded-xl border border-line bg-panel">
        <div class="flex items-center gap-2 border-b border-line px-5 py-3">
          <span class="text-xs text-mute">Swagger UI — OpenAPI 2.0</span>
          <span class="flex-1"></span>
          <button class="rounded-md border border-line bg-panel-2 px-2.5 py-1 text-xs hover:brightness-115" @click="swaggerUrl = '/swagger/index.html'">
            Reload
          </button>
        </div>
        <iframe :src="swaggerUrl" class="h-[560px] w-full border-0" title="Swagger UI"></iframe>
      </div>

      <div class="rounded-xl border border-line bg-panel p-5">
        <h3 class="m-0 text-sm font-semibold text-mute mb-3">Dynamically exposed routes</h3>
        <p v-if="!apis.length" class="text-mute text-sm">No registered APIs yet.</p>
        <div v-else class="space-y-3">
          <div v-for="a in apis" :key="a.id" class="rounded-lg border border-line bg-panel-2 p-3">
            <div class="flex items-center justify-between">
              <code class="text-sm text-accent-2">{{ a.base_path }}/*</code>
              <span class="rounded-full px-2 py-0.5 text-[10px] font-semibold" :class="a.requires_auth ? 'bg-accent/15 text-accent-2' : 'bg-panel-2 text-mute'">
                {{ a.requires_auth ? 'JWT' : 'open' }}
              </span>
            </div>
            <div class="mt-1 text-xs text-mute">{{ a.methods.join(', ') }} → {{ a.upstream }}</div>
          </div>
        </div>
        <p class="mt-4 text-xs leading-relaxed text-mute">
          The Swagger UI documents the Go API itself. Registered upstream APIs
          are exposed by the gateway at their base paths and listed here.
        </p>
      </div>
    </div>
  </div>
</template>