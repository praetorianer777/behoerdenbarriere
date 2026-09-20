import { screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AgencyPage } from './Agency'
import { Ranking } from './Ranking'
import { api } from '../api/client'
import { agencyDetail, agencyList, latestScan, stats } from '../test/fixtures'
import { renderPage } from '../test/render'

/**
 * zoll.de verlangt in seiner robots.txt 180 Sekunden Pause. Wir halten uns daran, und
 * heraus kommt ein Wert über eine einzige Seite — der im Ranking neben Werten über
 * hundert Seiten stand, als bedeute er dasselbe.
 */
describe('Vorläufige Werte', () => {
  beforeEach(() => {
    vi.spyOn(api, 'stats').mockResolvedValue(stats)
    vi.spyOn(api, 'latestScan').mockResolvedValue(latestScan)
  })
  afterEach(() => vi.restoreAllMocks())

  it('hängt den Hinweis an den Wert, nicht in eine Fußnote', async () => {
    vi.spyOn(api, 'agencies').mockResolvedValue({
      ...agencyList,
      items: [{ ...agencyList.items[0], pages: 1, provisional: true }, agencyList.items[1]],
    })
    renderPage(<Ranking />)

    const zeile = await screen.findByRole('row', { name: /Bundesregierung/ })
    expect(within(zeile).getByText('vorläufig')).toBeInTheDocument()
  })

  it('lässt belastbare Werte in Ruhe', async () => {
    vi.spyOn(api, 'agencies').mockResolvedValue(agencyList)
    renderPage(<Ranking />)

    await screen.findByRole('table')
    expect(screen.queryByText('vorläufig')).not.toBeInTheDocument()
  })

  it('sagt auf der Behördenseite, woran es liegt', async () => {
    vi.spyOn(api, 'agency').mockResolvedValue({ ...agencyDetail, pages: 1, provisional: true })
    renderPage(<AgencyPage />, { path: '/behoerde/:slug', route: '/behoerde/bundesregierung' })

    await screen.findByRole('heading', { level: 1, name: 'Bundesregierung' })
    const kasten = screen
      .getByRole('heading', { name: 'Dieser Wert ist vorläufig' })
      .closest('section')!
    expect(kasten).toHaveTextContent('1 geprüfte Seite')
    // Die Ursache ist die robots.txt der Behörde, nicht ihr Versäumnis — und erst
    // recht nicht etwas, das wir umgehen.
    expect(kasten).toHaveTextContent(/robots\.txt/)
    expect(kasten).toHaveTextContent(/umgehen das nicht/)
  })
})
