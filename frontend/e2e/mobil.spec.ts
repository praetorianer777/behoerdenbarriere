import { expect, test } from '@playwright/test'

import { routes, stubApi } from './fixtures'

test.beforeEach(async ({ page }) => {
  await stubApi(page)
})

/**
 * WCAG 1.4.10 verlangt, dass bei 320 CSS-Pixeln nicht in zwei Richtungen gescrollt
 * werden muss. Deutsche Behördennamen sind lang — "Bundesministerium für Wohnen,
 * Stadtentwicklung und Bauwesen" hat die Überschrift schon einmal über den Rand
 * geschoben —, deshalb wird gemessen und nicht angeschaut.
 */
for (const route of routes) {
  test(`${route} scrollt nicht quer`, async ({ page }, testInfo) => {
    await page.goto(route)
    await page.getByRole('heading', { level: 1 }).waitFor()

    const { documentWidth, viewport } = await page.evaluate(() => ({
      documentWidth: Math.ceil(document.documentElement.scrollWidth),
      viewport: window.innerWidth,
    }))

    expect(
      documentWidth,
      `${route} ist bei ${testInfo.project.name} ${documentWidth}px breit, Fenster ${viewport}px`,
    ).toBeLessThanOrEqual(viewport + 1)
  })
}

test('Rangliste wird auf dem Telefon zur Liste, nicht zur Schiebetabelle', async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name === 'desktop', 'gilt nur für schmale Fenster')

  await page.goto('/')
  await expect(page.getByRole('link', { name: /Bundesministerium für Wohnen/ })).toBeVisible()
  await expect(page.getByRole('table')).toBeHidden()
})

test('Bedienelemente sind groß genug zum Antippen', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile', 'gilt für Touch-Geräte')

  await page.goto('/')

  // 44 px ist die Größe, unter der Treffsicherheit spürbar nachlässt; WCAG 2.2 nennt
  // in 2.5.8 mindestens 24 px als Untergrenze.
  for (const name of ['Ranking', 'Überblick', 'Methodik']) {
    const box = await page.getByRole('link', { name, exact: true }).boundingBox()
    expect(box, `${name} hat keine Fläche`).not.toBeNull()
    expect(box!.height, `${name} ist ${box!.height}px hoch`).toBeGreaterThanOrEqual(44)
  }

  // Auf dem Telefon stehen die Filter eingeklappt. Was eingeklappt ist, hat keine
  // Fläche — erst aufklappen, dann messen.
  await page.getByRole('button', { name: /Filter und Sortierung/ }).click()
  const select = await page.getByLabel('Ebene').boundingBox()
  expect(select!.height).toBeGreaterThanOrEqual(44)
})

/**
 * Suchfeld und vier Auswahlen füllten bei 390 px den ganzen ersten Bildschirm: Man
 * öffnete das Ranking und sah kein Ranking.
 */
test('das Ranking beginnt auf dem Telefon im ersten Bildschirm', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name === 'desktop', 'gilt nur für schmale Fenster')

  await page.goto('/')
  const erste = page.locator('main ul li').first()
  await erste.waitFor()

  const box = await erste.boundingBox()
  const hoehe = page.viewportSize()!.height
  expect(box, 'die erste Behörde hat keine Fläche').not.toBeNull()
  expect(
    box!.y,
    `die erste Behörde beginnt bei ${Math.round(box!.y)}px, Fenster ${hoehe}px`,
  ).toBeLessThan(hoehe)
})

test('die Filter lassen sich auf dem Telefon aufklappen', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name === 'desktop', 'im breiten Fenster stehen sie immer offen')

  await page.goto('/')
  const schalter = page.getByRole('button', { name: /Filter und Sortierung/ })
  await expect(schalter).toHaveAttribute('aria-expanded', 'false')
  await expect(page.getByLabel('Behörde suchen')).toBeHidden()

  await schalter.click()
  await expect(schalter).toHaveAttribute('aria-expanded', 'true')
  await expect(page.getByLabel('Behörde suchen')).toBeVisible()
})
