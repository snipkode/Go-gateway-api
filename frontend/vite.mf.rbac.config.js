import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  define: { 'process.env.NODE_ENV': JSON.stringify('production') },
  base: '/admin/mf/rbac/',
  publicDir: false,
  plugins: [vue(), tailwindcss()],
  build: {
    outDir: 'dist/mf/rbac',
    emptyOutDir: true,
    cssCodeSplit: false,
    rollupOptions: {
      external: ['vue']
    },
    lib: {
      entry: 'src/micro/rbac/main.js',
      formats: ['es'],
      fileName: () => 'index.js',
      cssFileName: 'style'
    }
  }
})
