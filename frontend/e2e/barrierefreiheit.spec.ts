import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'

import { routes, stubApi } from './fixtures'

test.beforeEach(async ({ page }) => {
  await stubApi(page)
})

/**
 * Ein Monitor für Barrierefreiheit, der selbst durchfällt, ist wertlos. Geprüft wird
 * mit derselben Bibliothek und denselben Regeln, mit denen wir die Behörden prüfen —
 * hier im echten Browser, wo auch Kontraste messbar sind, die im jsdom-Test der
 * Komponenten niemand sehen kann.
 */
for (const route of routes) {
  test(`${route} erfüllt WCAG 2.1 AA`, async ({ page }) => {
    await page.goto(route)
    await page.getByRole('heading', { level: 1 }).waitFor()

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze()

    expect(
      results.violations.map((violation) => `${violation.id} (${violation.nodes.length})`),
    ).toEqual([])
  })
}

test('der Sprunglink führt zum Inhalt', async ({ page }) => {
  await page.goto('/')

  await page.keyboard.press('Tab')
  const skip = page.getByRole('link', { name: 'Zum Inhalt springen' })
  await expect(skip).toBeFocused()

  await skip.press('Enter')
  await expect(page).toHaveURL(/#inhalt$/)
})

test('die Rangliste lässt sich mit der Tastatur bedienen', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'die Tabelle gibt es nur im breiten Fenster')

  await page.goto('/')
  await page.getByRole('button', { name: /Score/ }).focus()
  await page.keyboard.press('Enter')

  await expect(page).toHaveURL(/sort=score_asc/)
})
