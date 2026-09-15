import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'

// In dev, /admin and /env.js proxy to the local main-api (docker stack), so
// the app is same-origin exactly like production behind Caddy.
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      registerType: 'prompt',
      devOptions: { enabled: false },
      manifest: {
        name: 'Bright Vintage Finds — Inventory',
        short_name: 'Inventory',
        description: 'Intake and inventory for the resell business',
        display: 'standalone',
        start_url: '/inventory',
        theme_color: '#3b2f2a',
        background_color: '#faf7f2',
      },
      workbox: {
        // Installable app shell only; API responses are never cached.
        navigateFallback: '/index.html',
        runtimeCaching: [],
        cleanupOutdatedCaches: true,
      },
    }),
  ],
  resolve: {
    // '/src' is root-relative, so no node:path / @types/node needed.
    alias: { '@': '/src' },
  },
  server: {
    proxy: {
      '/admin': 'http://localhost:3000',
      '/env.js': 'http://localhost:3000',
    },
  },
})
