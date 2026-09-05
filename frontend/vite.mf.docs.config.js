import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  define: { 'process.env.NODE_ENV': JSON.stringify('production') },
  base: '/admin/mf/docs/',
  publicDir: false,
  plugins: [vue(), tailwindcss()],
  build: {
    outDir: 'dist/mf/docs',
    emptyOutDir: true,
    cssCodeSplit: false,
    rollupOptions: {
      // Share the single Vue runtime from the shell (see vite.config.js /
      // index.html importmap). Externalising avoids a second Vue copy inside
      // each micro bundle, so reactive updates re-render across the console.
      external: ['vue']
    },
    lib: {
      entry: 'src/micro/docs/main.js',
      formats: ['es'],
      fileName: () => 'index.js',
      cssFileName: 'style'
    }
  }
})
