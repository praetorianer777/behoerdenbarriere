import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { axe } from 'vitest-axe'
import * as matchers from 'vitest-axe/matchers'

import { api } from '../api/client'
import { Dashboard } from '../pages/Dashboard'
import { Methodology } from '../pages/Methodology'
import { Ranking } from '../pages/Ranking'
import { AgencyPage } from '../pages/Agency'
import { agencyDetail, agencyList, latestScan, stats } from './fixtures'
import { renderPage } from './render'

expect.extend(matchers)

/**
 * Ein Monitor für Barrierefreiheit, der selbst durchfällt, ist wertlos. Geprüft wird
 * mit derselben Bibliothek, mit der wir die Behörden prüfen — hier auf dem gerenderten
 * DOM, in der CI zusätzlich auf der gebauten Seite im echten Browser.
 */
async function expectNoViolations(container: HTMLElement) {
  const results = await axe(container, {
    runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'] },
  })
  expect(results).toHaveNoViolations()
}

describe('Barrierefreiheit der eigenen Seiten', () => {
  beforeEach(() => {
    vi.spyOn(api, 'agencies').mockResolvedValue(agencyList)
    vi.spyOn(api, 'stats').mockResolvedValue(stats)
    vi.spyOn(api, 'agency').mockResolvedValue(agencyDetail)
    vi.spyOn(api, 'latestScan').mockResolvedValue(latestScan)
  })
  afterEach(() => vi.restoreAllMocks())

  it('Ranking', async () => {
    const { container } = renderPage(<Ranking />)
    await screen.findByRole('table')
    await expectNoViolations(container)
  })

  it('Behördenseite', async () => {
    const { container } = renderPage(<AgencyPage />, {
      path: '/behoerde/:slug',
      route: '/behoerde/bundesregierung',
    })
    await screen.findByRole('heading', { level: 1, name: 'Bundesregierung' })
    await expectNoViolations(container)
  })

  it('Überblick', async () => {
    const { container } = renderPage(<Dashboard />)
    await screen.findByRole('heading', { level: 1, name: 'Überblick' })
    await expectNoViolations(container)
  })

  it('Methodik', async () => {
    const { container } = render(<Methodology />)
    await expectNoViolations(container)
  })
})
