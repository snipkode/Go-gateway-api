import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  define: { 'process.env.NODE_ENV': JSON.stringify('production') },
  base: '/admin/mf/simulator/',
  publicDir: false,
  plugins: [vue(), tailwindcss()],
  build: {
    outDir: 'dist/mf/simulator',
    emptyOutDir: true,
    cssCodeSplit: false,
    lib: {
      entry: 'src/micro/simulator/main.js',
      formats: ['es'],
      fileName: () => 'index.js',
      cssFileName: 'style'
    }
  }
})
