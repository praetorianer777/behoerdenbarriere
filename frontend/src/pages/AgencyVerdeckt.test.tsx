import { screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AgencyPage } from './Agency'
import { Ranking } from './Ranking'
import { api } from '../api/client'
import { agencyDetail, agencyList, latestScan, stats } from '../test/fixtures'
import { renderPage } from '../test/render'

/**
 * Aachen bekam 91,2 und ein großes A, und zwei Bildschirme tiefer stand, dass auf 61
 * von 67 Seiten die Einwilligungsabfrage nicht zu schließen war. Ein sauber gebautes
 * Banner vor einer ungeprüften Seite ergibt eine gute Note.
 */
describe('Werte hinter einem Einwilligungsbanner', () => {
  beforeEach(() => {
    vi.spyOn(api, 'stats').mockResolvedValue(stats)
    vi.spyOn(api, 'latestScan').mockResolvedValue(latestScan)
  })
  afterEach(() => vi.restoreAllMocks())

  it('kennzeichnet den Wert schon im Ranking', async () => {
    vi.spyOn(api, 'agencies').mockResolvedValue({
      ...agencyList,
      items: [{ ...agencyList.items[0], obscured: true }, agencyList.items[1]],
    })
    renderPage(<Ranking />)

    const zeile = await screen.findByRole('row', { name: /Bundesregierung/ })
    expect(within(zeile).getByText('verdeckt')).toBeInTheDocument()
  })

  it('lässt Werte ohne Banner in Ruhe', async () => {
    vi.spyOn(api, 'agencies').mockResolvedValue(agencyList)
    renderPage(<Ranking />)

    await screen.findByRole('table')
    expect(screen.queryByText('verdeckt')).not.toBeInTheDocument()
  })

  it('sagt auf der Behördenseite, was das für den Wert heißt', async () => {
    vi.spyOn(api, 'agency').mockResolvedValue({ ...agencyDetail, obscured: true })
    renderPage(<AgencyPage />, { path: '/behoerde/:slug', route: '/behoerde/bundesregierung' })

    await screen.findByRole('heading', { level: 1, name: 'Bundesregierung' })
    const kasten = screen.getByRole('heading', { name: /Einwilligungsbanner/ }).closest('section')!
    expect(kasten).toHaveTextContent(/nicht schließen/)
    // Beide Richtungen: Ein gutes Ergebnis ist hier so wenig belastbar wie ein
    // schlechtes.
    expect(kasten).toHaveTextContent(/in beide Richtungen falsch/)
  })

  // Der kurze Aufkleber reicht für das Auge; wer ihn vorgelesen bekommt, braucht den
  // ganzen Satz.
  it('erklärt den Aufkleber für Screenreader', async () => {
    vi.spyOn(api, 'agency').mockResolvedValue({ ...agencyDetail, obscured: true })
    renderPage(<AgencyPage />, { path: '/behoerde/:slug', route: '/behoerde/bundesregierung' })

    await screen.findByRole('heading', { level: 1, name: 'Bundesregierung' })
    expect(screen.getByText(/überwiegend die Einwilligungsabfrage geprüft/)).toBeInTheDocument()
  })
})
