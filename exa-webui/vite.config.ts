import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'
import path from 'node:path'

const basePath = process.env.VITE_BASE_PATH || '/'
const apiProxyTarget = process.env.VITE_API_PROXY_TARGET || 'http://127.0.0.1:8197'
const baseNoSlash = basePath.replace(/\/$/, '') || ''

export default defineConfig({
  base: basePath,
  plugins: [
    vue(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['pwa-192x192.png', 'pwa-512x512.png'],
      manifest: {
        name: 'Exa',
        short_name: 'Exa',
        description: 'Personal expense tracker',
        theme_color: '#334155',
        background_color: '#0f172a',
        display: 'standalone',
        start_url: basePath,
        icons: [
          { src: 'pwa-192x192.png', sizes: '192x192', type: 'image/png', purpose: 'any maskable' },
          { src: 'pwa-512x512.png', sizes: '512x512', type: 'image/png', purpose: 'any maskable' },
        ],
      },
      workbox: {
        navigateFallback: basePath === '/' ? '/index.html' : `${baseNoSlash}/index.html`,
        runtimeCaching: [
          {
            urlPattern: ({ request }) =>
              request.destination === 'style' ||
              request.destination === 'script' ||
              request.destination === 'worker' ||
              request.destination === 'font',
            handler: 'CacheFirst',
            options: {
              cacheName: 'static-assets',
              expiration: { maxEntries: 80, maxAgeSeconds: 60 * 60 * 24 * 30 },
            },
          },
          {
            urlPattern: ({ url }) => url.pathname.includes('/api/'),
            handler: 'NetworkOnly',
          },
        ],
      },
      devOptions: { enabled: false },
    }),
  ],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  server: {
    host: true,
    port: Number(process.env.VITE_DEV_PORT || 5197),
    strictPort: true,
    allowedHosts: true,
    watch: {
      usePolling: process.env.CHOKIDAR_USEPOLLING === 'true',
    },
    hmr: {
      clientPort: Number(process.env.VITE_HMR_CLIENT_PORT || process.env.VITE_DEV_PORT || 5197),
    },
    proxy: {
      [`${baseNoSlash}/api`]: {
        target: apiProxyTarget,
        changeOrigin: true,
        rewrite: (p) => p.replace(new RegExp(`^${baseNoSlash}/api`), '/api'),
      },
      '/api': {
        target: apiProxyTarget,
        changeOrigin: true,
      },
      [`${baseNoSlash}/health`]: {
        target: apiProxyTarget,
        changeOrigin: true,
        rewrite: (p) => p.replace(new RegExp(`^${baseNoSlash}/health`), '/health'),
      },
      '/health': {
        target: apiProxyTarget,
        changeOrigin: true,
      },
    },
  },
})
