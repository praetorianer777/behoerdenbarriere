import { screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { Ranking } from './Ranking'
import { api } from '../api/client'
import { agencyList } from '../test/fixtures'
import { renderPage } from '../test/render'

/**
 * Das Ranking hat zwei Darstellungen: eine Tabelle im breiten Fenster, Karten im
 * schmalen. Die Hinweise am Wert hingen nur an der Tabelle — auf dem Telefon stand die
 * Zahl blank da, und das ist die Mehrheit der Lesenden.
 *
 * In jsdom wirkt kein CSS, hier stehen beide Darstellungen nebeneinander. Deshalb wird
 * ausdrücklich die Karte gesucht und nicht irgendein Treffer.
 */
function karte(name: RegExp): HTMLElement {
  const links = screen.getAllByRole('link', { name })
  const li = links.map((link) => link.closest('li')).find(Boolean)
  if (!li) throw new Error('keine Karte in der Telefonansicht gefunden')
  return li
}

describe('Telefonansicht des Rankings', () => {
  beforeEach(() => {
    vi.spyOn(api, 'stats').mockResolvedValue({} as never)
  })
  afterEach(() => vi.restoreAllMocks())

  it('trägt den Hinweis „vorläufig" auch auf der Karte', async () => {
    vi.spyOn(api, 'agencies').mockResolvedValue({
      ...agencyList,
      items: [{ ...agencyList.items[0], provisional: true }, agencyList.items[1]],
    })
    renderPage(<Ranking />)

    await screen.findAllByRole('link', { name: /Bundesregierung/ })
    expect(within(karte(/Bundesregierung/)).getByText('vorläufig')).toBeInTheDocument()
  })

  it('trägt den Hinweis „verdeckt" auch auf der Karte', async () => {
    vi.spyOn(api, 'agencies').mockResolvedValue({
      ...agencyList,
      items: [{ ...agencyList.items[0], obscured: true }, agencyList.items[1]],
    })
    renderPage(<Ranking />)

    await screen.findAllByRole('link', { name: /Bundesregierung/ })
    expect(within(karte(/Bundesregierung/)).getByText('verdeckt')).toBeInTheDocument()
  })

  it('lässt eine Karte ohne Einschränkung in Ruhe', async () => {
    vi.spyOn(api, 'agencies').mockResolvedValue(agencyList)
    renderPage(<Ranking />)

    await screen.findAllByRole('link', { name: /Bundesregierung/ })
    const inhalt = karte(/Bundesregierung/).textContent ?? ''
    expect(inhalt).not.toContain('vorläufig')
    expect(inhalt).not.toContain('verdeckt')
  })
})
