import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: '../internal/webapi/ui',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8090',
    },
  },
})
