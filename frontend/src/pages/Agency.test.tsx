import { screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AgencyPage } from './Agency'
import { api } from '../api/client'
import { agencyDetail, latestScan } from '../test/fixtures'
import { renderPage } from '../test/render'

function render() {
  return renderPage(<AgencyPage />, {
    path: '/behoerde/:slug',
    route: '/behoerde/bundesregierung',
  })
}

describe('Behördenseite', () => {
  beforeEach(() => {
    vi.spyOn(api, 'agency').mockResolvedValue(agencyDetail)
    vi.spyOn(api, 'latestScan').mockResolvedValue(latestScan)
  })
  afterEach(() => vi.restoreAllMocks())

  it('zeigt Score, Teilscores und Verlauf', async () => {
    render()

    expect(await screen.findByRole('heading', { level: 1, name: 'Bundesregierung' })).toBeInTheDocument()
    expect(screen.getByText(/Note C/)).toBeInTheDocument()

    const perceivable = screen.getByRole('heading', { name: 'Wahrnehmbar' }).parentElement!
    expect(within(perceivable).getByText('68')).toBeInTheDocument()
  })

  it('benennt Tempo und Hochrechnung', async () => {
    render()
    expect(await screen.findByText(/5 Punkte pro Monat/)).toBeInTheDocument()
    expect(screen.getByText(/Note A in etwa 3 Monate/)).toBeInTheDocument()
  })

  it('zeigt gefundene und behobene Verstöße', async () => {
    render()

    expect(await screen.findByRole('heading', { name: 'Verstöße nach Regel' })).toBeInTheDocument()
    expect(screen.getByText(/Mindestkontrast/)).toBeInTheDocument()

    const fixed = await screen.findByRole('heading', { name: /behoben/ })
    expect(fixed).toBeInTheDocument()
    expect(screen.getByText('Bilder brauchen eine Alternative')).toBeInTheDocument()
  })

  // Eine Seite, die nicht geladen werden konnte, ist kein Befund über Barrierefreiheit
  // und muss als das erkennbar sein.
  it('kennzeichnet nicht erreichbare Seiten', async () => {
    render()
    expect(await screen.findByText('nicht erreichbar')).toBeInTheDocument()
  })

  // Ein Banner, das sich nicht schließen ließ, macht den Befund zu einem Befund über
  // das Banner — und das muss auf der Seite stehen, nicht nur in den Daten.
  it('weist auf nicht schließbare Einwilligungsabfragen hin', async () => {
    vi.spyOn(api, 'latestScan').mockResolvedValue({
      ...latestScan,
      pages_blocked: 2,
      pages: [
        { ...latestScan.pages![0], consent: 'blocked' },
      ],
    })

    render()
    expect(await screen.findByText(/nicht schließen/)).toBeInTheDocument()
    expect(screen.getByText('hinter Einwilligungsabfrage')).toBeInTheDocument()
  })

  it('schweigt über Einwilligung, wenn keine im Weg stand', async () => {
    render()
    await screen.findByRole('heading', { name: 'Verstöße nach Regel' })
    expect(screen.queryByText(/nicht schließen/)).not.toBeInTheDocument()
  })

  it('kommt ohne Prüfung aus', async () => {
    vi.spyOn(api, 'agency').mockResolvedValue({
      ...agencyDetail,
      score: null,
      grade: undefined,
      scanned_at: undefined,
      latest_scan_id: undefined,
      history: [],
      trend: { direction: 'unknown', scans: 0 },
    })

    render()
    expect(await screen.findByText('noch nicht geprüft')).toBeInTheDocument()
    expect(screen.getByText('Diese Behörde wurde noch nicht geprüft.')).toBeInTheDocument()
    // Ohne Prüfung darf keine Hochrechnung behauptet werden.
    expect(screen.queryByText(/Note A in etwa/)).not.toBeInTheDocument()
  })
})
