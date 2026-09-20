import { screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { Drittanbieter } from './Drittanbieter'
import { api } from '../api/client'
import { thirdParties } from '../test/fixtures'
import { renderPage } from '../test/render'

describe('Drittanbieter', () => {
  beforeEach(() => {
    vi.spyOn(api, 'thirdParties').mockResolvedValue(thirdParties)
  })
  afterEach(() => vi.restoreAllMocks())

  it('trennt die Phasen und nennt den Anteil der Behörden', async () => {
    renderPage(<Drittanbieter />)

    await screen.findByRole('heading', { level: 1, name: 'Drittanbieter auf Behördenseiten' })
    expect(screen.getByRole('heading', { name: 'vor der Einwilligung' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'nach Zustimmung' })).toBeInTheDocument()

    // 48 von 120 Behörden sind 40 Prozent — die Zahl allein sagt nichts.
    const fonts = await screen.findByRole('row', { name: /Google Fonts/ })
    expect(within(fonts).getByText('48')).toBeInTheDocument()
    expect(within(fonts).getByText('40 %')).toBeInTheDocument()
  })

  it('zeigt den beobachteten Hostnamen neben der Einordnung', async () => {
    renderPage(<Drittanbieter />)
    const fonts = await screen.findByRole('row', { name: /Google Fonts/ })
    expect(within(fonts).getByText(/fonts\.gstatic\.com/)).toBeInTheDocument()
  })

  it('sagt, was die Zahlen nicht hergeben', async () => {
    renderPage(<Drittanbieter />)
    await screen.findByRole('heading', { level: 2, name: 'Was diese Zahlen nicht sagen' })
    expect(screen.getByText(/nicht, wer die Daten am Ende verarbeitet/)).toBeInTheDocument()
  })

  it('meldet fehlende Beobachtungen, statt leere Tabellen zu zeigen', async () => {
    vi.spyOn(api, 'thirdParties').mockResolvedValue({ items: [], scanned: 0 })
    renderPage(<Drittanbieter />)

    expect(await screen.findByText('Es liegen noch keine Beobachtungen vor.')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })
})
