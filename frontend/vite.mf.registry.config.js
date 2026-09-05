import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  define: { 'process.env.NODE_ENV': JSON.stringify('production') },
  base: '/admin/mf/registry/',
  publicDir: false,
  plugins: [vue(), tailwindcss()],
  build: {
    outDir: 'dist/mf/registry',
    emptyOutDir: true,
    cssCodeSplit: false,
    lib: {
      entry: 'src/micro/registry/main.js',
      formats: ['es'],
      fileName: () => 'index.js',
      cssFileName: 'style'
    }
  }
})
