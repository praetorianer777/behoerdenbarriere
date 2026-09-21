import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// Das Datum, an dem diese Fassung gebaut wurde. Die eigene Erklärung zur
// Barrierefreiheit nennt es als Datum der letzten Prüfung — zu Recht: Die CI prüft
// jede Fassung mit axe, in drei Breiten und mit der Tastatur, bevor sie gebaut wird.
const buildDate = new Date().toISOString().slice(0, 10)

export default defineConfig({
  define: { __BUILD_DATE__: JSON.stringify(buildDate) },
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    // Unit- und Komponententests liegen unter src/, die End-to-End-Tests unter e2e/
    // und laufen mit Playwright in einem echten Browser. Ohne diese Grenze versucht
    // Vitest, die Playwright-Dateien im jsdom auszuführen.
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
  },
})
