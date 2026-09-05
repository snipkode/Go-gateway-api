<script setup>
import { ref, watch, inject, computed, markRaw, shallowRef } from 'vue'

const props = defineProps({
  name: { type: String, required: true },
  init: { type: String, default: '' }
})

const mfManifest = inject('mfManifest', ref([]))
const comp = shallowRef(null)
const err = ref('')

const entry = computed(() => mfManifest.value?.find((x) => x.name === props.name))

async function load() {
  err.value = ''
  comp.value = null
  const m = entry.value
  if (!m) {
    err.value = `micro-frontend "${props.name}" is not in the manifest`
    return
  }
  try {
    if (m.css && !window.__mfCssLoaded?.[m.css]) {
      window.__mfCssLoaded = window.__mfCssLoaded || {}
      const l = document.createElement('link')
      l.rel = 'stylesheet'
      l.href = m.css
      document.head.appendChild(l)
      window.__mfCssLoaded[m.css] = true
    }
    comp.value = markRaw((await import(/* @vite-ignore */ m.js)).default)
  } catch (e) {
    err.value = `failed to load "${props.name}": ${e.message}`
  }
}

watch([() => props.name, () => entry.value?.js], load, { immediate: true })
</script>

<template>
  <div>
    <div v-if="err" class="rounded-[18px] bg-panel p-6 text-center text-bad shadow-sm">{{ err }}</div>
    <component :is="comp" v-else-if="comp" :init="init" />
    <div v-else class="flex items-center justify-center gap-2 py-16 text-mute">
      <span class="h-4 w-4 animate-spin rounded-full border-2 border-mute/30 border-t-accent"></span>
      loading {{ name }}…
    </div>
  </div>
</template>