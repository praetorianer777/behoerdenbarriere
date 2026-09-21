import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { Ranking } from './Ranking'
import { api } from '../api/client'
import { agencyList } from '../test/fixtures'
import { renderPage } from '../test/render'

/**
 * Bei 390 px füllten Suchfeld und vier Auswahlen den ganzen ersten Bildschirm: Man
 * öffnete das Ranking und sah kein Ranking. Ab Tablettbreite steht der Kasten offen,
 * die Schaltfläche verschwindet — das entscheidet CSS, die Mechanik steht hier.
 */
describe('Filter im Ranking', () => {
  beforeEach(() => {
    vi.spyOn(api, 'agencies').mockResolvedValue(agencyList)
  })
  afterEach(() => vi.restoreAllMocks())

  it('beginnt zugeklappt und lässt sich aufklappen', async () => {
    const user = userEvent.setup()
    renderPage(<Ranking />)

    const schalter = await screen.findByRole('button', { name: /Filter und Sortierung/ })
    expect(schalter).toHaveAttribute('aria-expanded', 'false')

    await user.click(schalter)
    expect(schalter).toHaveAttribute('aria-expanded', 'true')
  })

  // Eine kurze Liste ohne sichtbaren Grund sieht aus wie eine leere Datenbank.
  it('steht offen, wenn schon gefiltert wurde, und sagt wie oft', async () => {
    renderPage(<Ranking />, { route: '/?level=bund&state=Bayern' })

    const schalter = await screen.findByRole('button', { name: /Filter und Sortierung \(2\)/ })
    expect(schalter).toHaveAttribute('aria-expanded', 'true')
  })

  it('zählt die Sortierung nicht als Filter', async () => {
    renderPage(<Ranking />, { route: '/?sort=scanned' })

    // Ohne Zähler im Namen: Sortieren ist kein Filtern.
    // Ein String als Name trifft nur den ganzen Namen — mit Zähler passt er nicht.
    const schalter = await screen.findByRole('button', { name: 'Filter und Sortierung' })
    expect(schalter).toHaveAttribute('aria-expanded', 'false')
  })

  it('beschriftet den Kasten für Hilfsmittel', async () => {
    const schalter = await (async () => {
      renderPage(<Ranking />)
      return screen.findByRole('button', { name: /Filter und Sortierung/ })
    })()
    expect(schalter).toHaveAttribute('aria-controls', 'filter')
  })
})
