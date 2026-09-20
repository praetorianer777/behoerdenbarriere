import { defineConfig, devices } from '@playwright/test'

// Die Tests laufen gegen die gebaute Seite, nicht gegen den Entwicklungsserver: was
// ausgeliefert wird, ist die Version mit gebündeltem CSS und nachgeladenem Diagramm,
// und genau daran hängen die Layoutfragen.
const port = 4173
const baseURL = process.env.E2E_BASE_URL ?? `http://localhost:${port}`

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : [['list']],
  use: {
    baseURL,
    trace: 'retain-on-failure',
  },
  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'] } },
    { name: 'mobile', use: { ...devices['Pixel 5'] } },
    // 320 px ist die Breite, auf die sich WCAG 1.4.10 bezieht — das kleinste Gerät,
    // das noch in Gebrauch ist.
    {
      name: 'schmal',
      use: { ...devices['Desktop Chrome'], viewport: { width: 320, height: 800 }, isMobile: false },
    },
  ],
  webServer: process.env.E2E_BASE_URL
    ? undefined
    : {
        command: `npm run build && npx vite preview --port ${port} --strictPort`,
        url: baseURL,
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
      },
})
