import { screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AgencyPage } from './Agency'
import { api } from '../api/client'
import { agencyDetail } from '../test/fixtures'
import { renderPage } from '../test/render'

/**
 * Direkt nach der Installation ist das der Normalfall: Behörden sind eingespielt,
 * geprüft wurde noch keine. Wer dann im Ranking auf einen Namen klickte, bekam eine
 * weiße Seite — die leere Historie kam als `null`, das Diagramm las darauf eine Länge.
 */
const ungeprueft = {
  ...agencyDetail,
  score: null,
  grade: undefined,
  pages: 0,
  scanned_at: undefined,
  latest_scan_id: undefined,
  subscores: { perceivable: null, operable: null, understandable: null, robust: null },
  trend: { direction: 'unknown' as const, scans: 0 },
}

describe('Behörde ohne Prüfung', () => {
  beforeEach(() => {
    vi.spyOn(api, 'latestScan').mockRejectedValue(new Error('404 Not Found'))
  })
  afterEach(() => vi.restoreAllMocks())

  async function render(history: unknown) {
    vi.spyOn(api, 'agency').mockResolvedValue({ ...ungeprueft, history } as never)
    renderPage(<AgencyPage />, { path: '/behoerde/:slug', route: '/behoerde/bundesregierung' })
    return screen.findByRole('heading', { level: 1, name: 'Bundesregierung' })
  }

  it('zeigt die Behörde, auch wenn die Historie fehlt', async () => {
    await render(null)

    expect(screen.getByText('nicht geprüft')).toBeInTheDocument()
    // Bis hierher kommt auch die kaputte Fassung: Das Diagramm wird nachgeladen und
    // stürzt erst danach ab. Also abwarten, bis es da ist — sonst prüft der Test die
    // Sekunde vor dem Fehler.
    expect(await screen.findByText(/mindestens zwei Prüfungen/)).toBeInTheDocument()
    // Keine Fehlergrenze in Sicht: Die Seite trägt, sie stürzt nicht.
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('kommt auch mit einer leeren Liste zurecht', async () => {
    await render([])
    // Das Diagramm wird nachgeladen, deshalb erst suchen, wenn es da ist.
    expect(await screen.findByText(/mindestens zwei Prüfungen/)).toBeInTheDocument()
  })

  it('erfindet keine Prüfergebnisse', async () => {
    await render(null)

    expect(screen.queryByRole('heading', { name: 'Verstöße nach Regel' })).not.toBeInTheDocument()
    expect(
      screen.queryByRole('heading', { name: 'Erklärung zur Barrierefreiheit' }),
    ).not.toBeInTheDocument()
  })
})
