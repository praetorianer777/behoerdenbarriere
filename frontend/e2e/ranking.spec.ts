import { expect, test } from '@playwright/test'

import { stubApi } from './fixtures'

test.beforeEach(async ({ page }) => {
  await stubApi(page)
})

test('zeigt Behörden mit Note und führt zur Detailseite', async ({ page }) => {
  await page.goto('/')

  await expect(page.getByRole('heading', { level: 1 })).toContainText('Wie barrierefrei')
  await expect(page.getByText('2 Behörden gefunden')).toBeVisible()

  await page
    .getByRole('link', { name: /Bundesministerium für Wohnen/ })
    .first()
    .click()

  await expect(page).toHaveURL(/\/behoerde\/bmwsb$/)
  await expect(page.getByRole('heading', { level: 1 })).toContainText(
    'Bundesministerium für Wohnen',
  )
})

// Eine ungeprüfte Behörde darf nicht wie eine mit null Punkten aussehen — das ist der
// Unterschied zwischen "niemand hat nachgesehen" und "schlecht".
test('unterscheidet ungeprüft von schlecht', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByText('nicht geprüft').locator('visible=true').first()).toBeVisible()
})

test('Filter landen in der Adresse und überleben einen Neuladen', async ({ page }) => {
  await page.goto('/')

  await page.getByLabel('Ebene').selectOption('kommune')
  await expect(page).toHaveURL(/level=kommune/)

  await page.reload()
  await expect(page.getByLabel('Ebene')).toHaveValue('kommune')
})

test('nennt Tempo und Hochrechnung auf der Detailseite', async ({ page }) => {
  await page.goto('/behoerde/bmwsb')

  await expect(page.getByText(/5 Punkte pro Monat/)).toBeVisible()
  await expect(page.getByText(/Note A in etwa 3 Monate/)).toBeVisible()
})

// Ein Banner, das sich nicht schließen ließ, macht den Befund zu einem Befund über das
// Banner. Wer den Score liest, muss das auf der Seite sehen.
test('weist auf nicht schließbare Einwilligungsabfragen hin', async ({ page }) => {
  await page.goto('/behoerde/bmwsb')
  await expect(page.getByText(/nicht schließen/)).toBeVisible()
})

test('zeigt beide Bewertungen nebeneinander', async ({ page }) => {
  await page.goto('/behoerde/bmwsb')

  await expect(page.getByRole('heading', { name: /Lighthouse/ })).toBeVisible()
  await expect(page.getByText(/ganz oder gar nicht/)).toBeVisible()
})

test('Verlauf steht auch als Tabelle bereit', async ({ page }) => {
  await page.goto('/behoerde/bmwsb')

  // Das Diagramm ist ein SVG; für Screenreader und Tastatur zählt die Tabelle.
  await page.getByText('Werte als Tabelle').click()
  await expect(page.getByRole('cell', { name: '70,5' })).toBeVisible()
})
