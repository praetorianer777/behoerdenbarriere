import { screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '../api/client'
import { Dashboard } from './Dashboard'
import { stats } from '../test/fixtures'
import { renderPage } from '../test/render'

describe('Überblick', () => {
  beforeEach(() => {
    vi.spyOn(api, 'stats').mockResolvedValue(stats)
  })

  // Ein Durchschnitt neben der Zahl aller Behörden liest sich als Aussage über alle:
  // Das Saarland stand mit acht Behörden und einer Null da, geprüft war eine.
  it('sagt, auf wie vielen Behörden ein Durchschnitt beruht', async () => {
    renderPage(<Dashboard />)

    const zeile = (await screen.findByRole('rowheader', { name: 'Schleswig-Holstein' }))
      .parentElement!
    const zellen = Array.from(zeile.querySelectorAll('td')).map((cell) => cell.textContent)
    expect(zellen).toEqual(['4', '0', 'nicht geprüft'])
  })

  it('nennt die Barrieren auf Deutsch', async () => {
    renderPage(<Dashboard />)
    expect(
      await screen.findByRole('rowheader', {
        name: 'Zu wenig Kontrast zwischen Text und Hintergrund',
      }),
    ).toBeInTheDocument()
  })
})
