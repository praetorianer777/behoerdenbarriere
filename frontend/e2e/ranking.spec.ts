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

// Eine Zahl, die nicht sagt, woraus sie besteht, ist eine Behauptung — und die
// Begründung muss auch auf dem Telefon erreichbar sein, nicht nur am Schreibtisch.
test('begründet den Wert je Seite', async ({ page }) => {
  await page.goto('/behoerde/bmwsb')

  await expect(page.getByRole('heading', { name: 'Was am meisten bringt' })).toBeVisible()
  await expect(page.getByText(/\+16,1 Punkte/)).toBeVisible()

  // Aufklappen über die Zusammenfassung, so wie es auch eine Nutzerin täte.
  await page.locator('summary').filter({ hasText: 'Startseite' }).first().click()

  await expect(page.getByRole('table', { name: /Befunde dieser Seite/ })).toBeVisible()
  await expect(page.getByRole('cell', { name: '78 %' })).toBeVisible()
  await expect(page.getByRole('cell', { name: '+21,4' })).toBeVisible()
})

// Wer von 426 Behörden die Pflichtangaben verlangt, muss die eigenen vorzeigen —
// und zwar von jeder Seite aus erreichbar, wie das Gesetz es für Behörden verlangt.
test('führt die eigenen Rechtstexte im Fuß jeder Seite', async ({ page }) => {
  for (const route of ['/', '/methodik']) {
    await page.goto(route)
    const fuss = page.getByRole('navigation', { name: 'Rechtliches' })
    await expect(fuss.getByRole('link', { name: 'Impressum' })).toBeVisible()
    await expect(fuss.getByRole('link', { name: 'Datenschutz' })).toBeVisible()
    await expect(fuss.getByRole('link', { name: 'Erklärung zur Barrierefreiheit' })).toBeVisible()
  }
})

test('die eigene Erklärung nennt alle Pflichtangaben', async ({ page }) => {
  await page.goto('/barrierefreiheit')

  // Dieselben sechs Punkte, die wir bei Behörden prüfen.
  await expect(page.getByText(/teilweise vereinbar/)).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Nicht barrierefreie Inhalte' })).toBeVisible()
  await expect(page.getByText(/erstellt und zuletzt überprüft/)).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Barrieren melden' })).toBeVisible()
  await expect(page.getByText(/Schlichtungsstelle nach § 16 BGG/)).toBeVisible()
})

// Ein erfundenes Impressum wäre schlimmer als ein fehlendes, deshalb sagt die Seite
// selbst, solange die Angaben fehlen.
test('weist auf fehlende Betreiberangaben hin', async ({ page }) => {
  await page.goto('/impressum')
  await expect(page.getByRole('alert')).toContainText('noch nicht vollständig')
})

test('zeigt die Prüfung der Erklärung zur Barrierefreiheit', async ({ page }) => {
  await page.goto('/behoerde/bmwsb')

  await expect(page.getByRole('heading', { name: 'Erklärung zur Barrierefreiheit' })).toBeVisible()
  await expect(page.getByText(/5 von 6 Pflichtangaben/)).toBeVisible()
  // Was fehlt, muss benannt sein — nicht nur, dass etwas fehlt.
  await expect(page.getByText('Hinweis auf das Schlichtungsverfahren')).toBeVisible()
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

test('nennt die Drittanbieter einer Behörde mit dem Zeitpunkt', async ({ page }) => {
  await page.goto('/behoerde/bmwsb')

  await expect(page.getByRole('heading', { name: 'Eingebundene Drittanbieter' })).toBeVisible()
  await expect(page.getByText('fonts.gstatic.com')).toBeVisible()
  // Der Zeitpunkt ist der Befund, nicht der bloße Host.
  await expect(page.getByRole('heading', { name: 'vor der Einwilligung' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'nach Zustimmung' })).toBeVisible()
})

test('führt die bundesweite Auswertung der Drittanbieter', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('link', { name: 'Drittanbieter' }).click()

  await expect(
    page.getByRole('heading', { level: 1, name: 'Drittanbieter auf Behördenseiten' }),
  ).toBeVisible()
  await expect(page.getByRole('row', { name: /Google Fonts/ })).toContainText('40 %')
  await expect(page.getByRole('heading', { name: 'Was diese Zahlen nicht sagen' })).toBeVisible()
})

test('zeigt die DNS-Einträge einer Behörde mit dem Rohwert', async ({ page }) => {
  await page.goto('/behoerde/bmwsb')

  await expect(page.getByRole('heading', { name: 'Wohin die E-Mail geht' })).toBeVisible()
  await expect(page.getByText('mx1.bund.de')).toBeVisible()
  // Der SPF-Eintrag darf nicht als Mail-Hosting gelesen werden.
  await expect(page.getByText(/nicht, wo die Postfächer liegen/)).toBeVisible()
})

test('führt die bundesweite Auswertung der E-Mail-Anbieter', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('link', { name: 'E-Mail' }).click()

  await expect(
    page.getByRole('heading', { level: 1, name: 'Wohin die Post der Behörden geht' }),
  ).toBeVisible()
  await expect(page.getByRole('row', { name: /Bayern/ })).toContainText('50 %')
  await expect(page.getByRole('heading', { name: 'Was ein MX-Eintrag nicht sagt' })).toBeVisible()
})
