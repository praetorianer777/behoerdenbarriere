import { screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { EMail } from './EMail'
import { api } from '../api/client'
import { mailSummary } from '../test/fixtures'
import { renderPage } from '../test/render'

describe('E-Mail-Auswertung', () => {
  beforeEach(() => {
    vi.spyOn(api, 'mail').mockResolvedValue(mailSummary)
  })
  afterEach(() => vi.restoreAllMocks())

  it('nennt den Anteil der Behörden bei US-Anbietern', async () => {
    renderPage(<EMail />)

    await screen.findByRole('heading', { level: 1, name: 'Wohin die Post der Behörden geht' })
    // 36 Microsoft + 10 Google von 120 sind 38 Prozent.
    expect(screen.getByText('46')).toBeInTheDocument()
    expect(screen.getByText(/\(38 %\)/)).toBeInTheDocument()
  })

  it('zeigt die Länder auch als Tabelle, nicht nur als Kacheln', async () => {
    renderPage(<EMail />)

    const bayern = await screen.findByRole('row', { name: /Bayern/ })
    expect(within(bayern).getByText('12')).toBeInTheDocument()
    expect(within(bayern).getByText('6 (50 %)')).toBeInTheDocument()
  })

  it('trennt Anbieter mit Sitz in den USA von den übrigen', async () => {
    renderPage(<EMail />)

    const microsoft = await screen.findByRole('row', { name: /Microsoft 365/ })
    expect(within(microsoft).getByText('USA')).toBeInTheDocument()

    const eigen = screen.getByRole('row', { name: /Eigener Betrieb/ })
    expect(within(eigen).queryByText('USA')).not.toBeInTheDocument()
  })

  it('sagt, was ein MX-Eintrag nicht hergibt', async () => {
    renderPage(<EMail />)

    await screen.findByRole('heading', { level: 2, name: 'Was ein MX-Eintrag nicht sagt' })
    expect(screen.getByText(/Ein SPF-Eintrag ist kein Mail-Hosting/)).toBeInTheDocument()
    expect(screen.getByText(/nicht, wer sie am Ende liest/)).toBeInTheDocument()
  })

  it('meldet fehlende Abfragen, statt Nullen zu zeigen', async () => {
    vi.spyOn(api, 'mail').mockResolvedValue({
      total: 0,
      by_provider: [],
      by_state: [],
      by_level: [],
    })
    renderPage(<EMail />)

    expect(await screen.findByText('Es liegen noch keine Abfragen vor.')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })
})
