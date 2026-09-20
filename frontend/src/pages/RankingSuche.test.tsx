import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { Ranking } from './Ranking'
import { api } from '../api/client'
import { agencyList, stats } from '../test/fixtures'
import { renderPage } from '../test/render'

/**
 * Beim Tippen hat die Liste geflackert: Jeder Tastendruck war eine Abfrage, und
 * während sie lief, verschwand die Tabelle. Sechs Buchstaben, sechs Sprünge.
 */
describe('Suche im Ranking', () => {
  beforeEach(() => {
    vi.spyOn(api, 'stats').mockResolvedValue(stats)
  })
  afterEach(() => vi.restoreAllMocks())

  it('fragt erst, wenn das Tippen aufhört', async () => {
    const abfrage = vi.spyOn(api, 'agencies').mockResolvedValue(agencyList)
    const user = userEvent.setup()
    renderPage(<Ranking />)

    await screen.findByRole('table')
    abfrage.mockClear()

    await user.type(screen.getByLabelText('Behörde suchen'), 'Aachen')

    // Eine Abfrage für das getippte Wort, nicht eine je Buchstabe.
    await waitFor(() => expect(abfrage).toHaveBeenCalledTimes(1))
    expect(abfrage.mock.calls[0][0]).toMatchObject({ q: 'Aachen' })
  })

  it('lässt die bisherige Liste stehen, solange die neue lädt', async () => {
    let antworten: (() => void) | undefined
    vi.spyOn(api, 'agencies').mockImplementation(async (query) => {
      if (!query.q) return agencyList
      // Die zweite Abfrage bleibt hängen — so sieht der Moment aus, in dem die
      // Tabelle bisher verschwand.
      await new Promise<void>((resolve) => {
        antworten = resolve
      })
      return { ...agencyList, items: [agencyList.items[0]], total: 1 }
    })

    const user = userEvent.setup()
    renderPage(<Ranking />)
    await screen.findByRole('table')

    await user.type(screen.getByLabelText('Behörde suchen'), 'Aachen')

    // Während die neue Liste unterwegs ist, steht die alte noch da — und ist als
    // „wird geladen" ausgewiesen.
    await waitFor(() => expect(screen.getByRole('table')).toBeInTheDocument())
    await waitFor(() => {
      const busy = document.querySelector('[aria-busy="true"]')
      expect(busy).not.toBeNull()
      expect(within(busy as HTMLElement).getByRole('table')).toBeInTheDocument()
    })

    antworten?.()
    await waitFor(() => expect(document.querySelector('[aria-busy="true"]')).toBeNull())
  })

  it('übernimmt den Suchtext in die Adresse', async () => {
    vi.spyOn(api, 'agencies').mockResolvedValue(agencyList)
    const user = userEvent.setup()
    renderPage(<Ranking />)
    await screen.findByRole('table')

    await user.type(screen.getByLabelText('Behörde suchen'), 'Kiel')

    await waitFor(() => expect(window.location.search).toBe(''))
    // Der MemoryRouter schreibt nicht in die Adresszeile des Browsers; geprüft wird
    // deshalb am Feld selbst, dass der Wert erhalten bleibt.
    expect(screen.getByLabelText('Behörde suchen')).toHaveValue('Kiel')
  })
})
