import { screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AgencyPage } from './Agency'
import { api } from '../api/client'
import { agencyDetail } from '../test/fixtures'
import { renderPage } from '../test/render'

/**
 * Bot-Schutz nimmt rund zwanzig Behörden aus dem Ranking. Ohne Begründung sähe das aus
 * wie „noch nicht drangekommen", und der Verdacht fiele auf die Behörde.
 */
describe('Behörde hinter einer Sperre', () => {
  beforeEach(() => {
    vi.spyOn(api, 'latestScan').mockRejectedValue(new Error('404 Not Found'))
    vi.spyOn(api, 'agency').mockResolvedValue({
      ...agencyDetail,
      score: null,
      grade: undefined,
      pages: 0,
      scanned_at: undefined,
      latest_scan_id: undefined,
      history: [],
      trend: { direction: 'unknown' as const, scans: 0 },
      failure: { reason: 'Bot-Schutz: Link11 - CAPTCHA', at: '2026-09-20T04:00:00Z' },
    })
  })
  afterEach(() => vi.restoreAllMocks())

  async function render() {
    renderPage(<AgencyPage />, { path: '/behoerde/:slug', route: '/behoerde/bundesregierung' })
    return screen.findByRole('heading', { level: 1, name: 'Bundesregierung' })
  }

  it('nennt den Grund, statt nur nichts zu zeigen', async () => {
    await render()

    expect(
      screen.getByRole('heading', { name: 'Diese Website konnte nicht geprüft werden' }),
    ).toBeInTheDocument()
    // Das Datum steht auch anderswo auf der Seite; hier zählt der Kasten.
    const kasten = screen
      .getByRole('heading', { name: 'Diese Website konnte nicht geprüft werden' })
      .closest('section')!
    expect(kasten).toHaveTextContent('Link11 - CAPTCHA')
    expect(kasten).toHaveTextContent('20.09.2026')
  })

  it('sagt dazu, dass das kein Urteil über die Behörde ist', async () => {
    await render()
    expect(screen.getByText(/keine Aussage über die Barrierefreiheit/)).toBeInTheDocument()
  })

  it('erfindet keine Note', async () => {
    await render()
    expect(screen.getByText('nicht geprüft')).toBeInTheDocument()
    expect(screen.queryByText(/Note [A-F]/)).not.toBeInTheDocument()
  })
})
