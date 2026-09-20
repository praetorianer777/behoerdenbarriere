/// <reference types="vitest/config" />
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
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
