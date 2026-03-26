import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  // Relative asset URLs so the SPA works at site root or under a path (e.g. /tracelog/).
  base: './',
  plugins: [svelte()],
  build: {
    outDir: '../internal/hub/dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8090',
        changeOrigin: true,
      },
    },
  },
})
