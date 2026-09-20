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

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Bundesregierung' }),
    ).toBeInTheDocument()
    expect(screen.getByText(/Note C/)).toBeInTheDocument()

    const perceivable = screen.getByRole('heading', {
      name: 'Wahrnehmbar',
    }).parentElement!
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
    // Derselbe Befund steht auch in der Begründung; hier zählt die Regelliste.
    expect(screen.getAllByText(/Zu wenig Kontrast/).length).toBeGreaterThan(0)

    const fixed = await screen.findByRole('heading', { name: /behoben/ })
    expect(fixed).toBeInTheDocument()
    expect(screen.getByText('Bild ohne Alternativtext')).toBeInTheDocument()
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
      pages: [{ ...latestScan.pages![0], consent: 'blocked' }],
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

  // Zwei Zahlen nebeneinander helfen nur, wenn dabeisteht, dass sie verschieden
  // rechnen — sonst liest man den Unterschied als Fehler.
  it('zeigt den Lighthouse-Wert mit Einordnung', async () => {
    render()

    expect(await screen.findByRole('heading', { name: /Lighthouse/ })).toBeInTheDocument()
    expect(screen.getByText('97')).toBeInTheDocument()
    expect(screen.getByText(/ganz oder gar nicht/)).toBeInTheDocument()
  })

  it('schweigt über Lighthouse, wenn nichts gemessen wurde', async () => {
    vi.spyOn(api, 'agency').mockResolvedValue({ ...agencyDetail, lighthouse_score: undefined })

    render()
    await screen.findByRole('heading', { level: 1, name: 'Bundesregierung' })
    expect(screen.queryByRole('heading', { name: /Lighthouse/ })).not.toBeInTheDocument()
  })

  // Eine Zahl, die nicht sagt, woraus sie besteht, ist eine Behauptung.
  it('begründet den Wert je Seite', async () => {
    render()

    await screen.findByRole('heading', { name: 'Begründung je Seite' })
    expect(screen.getByText('Was am meisten bringt')).toBeInTheDocument()
    // Was das Beheben zurückgibt, ist die Zahl, mit der jemand etwas anfangen kann.
    expect(screen.getByText(/\+16,1 Punkte/)).toBeInTheDocument()
    expect(screen.getByText(/78 %/)).toBeInTheDocument()
  })

  // Die Erklärung ist eine Rechtspflicht, keine Kennzahl — sie muss als eigener Punkt
  // erscheinen, mit dem, was fehlt.
  it('prüft die Erklärung zur Barrierefreiheit', async () => {
    render()

    await screen.findByRole('heading', { name: 'Erklärung zur Barrierefreiheit' })
    expect(screen.getByText(/5 von 6 Pflichtangaben/)).toBeInTheDocument()
    expect(screen.getByText('Hinweis auf das Schlichtungsverfahren')).toBeInTheDocument()
    // Der Fundort steht dabei, damit man dem Befund widersprechen kann.
    expect(screen.getAllByText(/teilweise vereinbar/).length).toBeGreaterThan(0)
  })

  it('benennt eine fehlende Erklärung als Rechtsverstoß', async () => {
    vi.spyOn(api, 'latestScan').mockResolvedValue({
      ...latestScan,
      statement: { state: 'missing' as const, findings: [] },
    })

    render()
    expect(
      await screen.findByText(/keine Erklärung zur Barrierefreiheit gefunden/),
    ).toBeInTheDocument()
    expect(screen.getByText(/§ 12b/)).toBeInTheDocument()
  })

  // Gesehen, aber nicht lesbar: Das darf nicht als "hat keine" erscheinen.
  it('unterscheidet gesperrt von fehlend', async () => {
    vi.spyOn(api, 'latestScan').mockResolvedValue({
      ...latestScan,
      statement: {
        state: 'unreadable' as const,
        url: 'https://www.rki.de/DE/Service/Barrierefreiheit/barrierefreiheit_node.html',
        findings: [],
      },
    })

    render()
    expect(await screen.findByText(/nicht abrufen/)).toBeInTheDocument()
    expect(
      screen.queryByText(/keine Erklärung zur Barrierefreiheit gefunden/),
    ).not.toBeInTheDocument()
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
