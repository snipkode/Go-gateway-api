import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  base: '/admin/',
  plugins: [vue(), tailwindcss()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:18080',
      '/swagger': 'http://localhost:18080',
      '/admin/mf-manifest.json': 'http://localhost:18080',
      '/admin/mf/': 'http://localhost:18080',
      // Registered-API base paths (Simulator tester in dev). Add any new
      // base_path you register while developing.
      '/idp': 'http://localhost:18080',
      '/ext': 'http://localhost:18080',
      '/orders': 'http://localhost:18080'
    }
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    rollupOptions: {
      // Single shared Vue runtime (served via importmap at /admin/mf/vue.js).
      // Shell + every micro resolve to the same module instance so reactive
      // state updates re-render across the whole console.
      external: ['vue']
    }
  }
})
