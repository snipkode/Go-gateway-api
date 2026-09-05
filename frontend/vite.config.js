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
      '/admin/mf/': 'http://localhost:18080'
    }
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets'
  }
})