import { screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { axe } from 'vitest-axe'

import { Ranking } from './Ranking'
import { api } from '../api/client'
import { agencyList } from '../test/fixtures'
import { renderPage } from '../test/render'

/**
 * Im Ranking steht neben dem Wert nur das Wort. Wer nie eine Behördenseite öffnet,
 * sieht einen Aufkleber und muss raten — und die naheliegenden Vermutungen sind beide
 * falsch: weder ist die Behörde in Verzug, noch sind wir noch nicht fertig.
 */
describe('Legende im Ranking', () => {
  beforeEach(() => {
    vi.spyOn(api, 'stats').mockResolvedValue({} as never)
  })
  afterEach(() => vi.restoreAllMocks())

  function mitHinweisen(over: { provisional?: boolean; obscured?: boolean }) {
    vi.spyOn(api, 'agencies').mockResolvedValue({
      ...agencyList,
      items: [{ ...agencyList.items[0], ...over }, agencyList.items[1]],
    })
  }

  it('erklärt „vorläufig" samt Ursache', async () => {
    mitHinweisen({ provisional: true })
    renderPage(<Ranking />)

    expect(await screen.findByText(/weniger als fünf Seiten/)).toBeInTheDocument()
    // Die Ursache ist die robots.txt der Behörde, nicht ihr Versäumnis.
    expect(screen.getByText(/robots\.txt/)).toBeInTheDocument()
  })

  it('erklärt „verdeckt" samt Ursache', async () => {
    mitHinweisen({ obscured: true })
    renderPage(<Ranking />)

    expect(await screen.findByText(/Abfrage nach Cookies nicht schließen/)).toBeInTheDocument()
  })

  // Sonst steht der Textblock auch unter den Listen, in denen kein einziger Wert
  // gekennzeichnet ist — und wird dann überall überlesen.
  it('erklärt nur, was in der Liste vorkommt', async () => {
    mitHinweisen({ provisional: true })
    renderPage(<Ranking />)

    await screen.findByText(/weniger als fünf Seiten/)
    expect(screen.queryByText(/Abfrage nach Cookies/)).not.toBeInTheDocument()
  })

  it('schweigt, wenn kein Wert gekennzeichnet ist', async () => {
    vi.spyOn(api, 'agencies').mockResolvedValue(agencyList)
    renderPage(<Ranking />)

    await screen.findByRole('table')
    expect(screen.queryByRole('heading', { name: 'Die Hinweise an den Werten' })).toBeNull()
  })

  it('ist selbst barrierefrei', async () => {
    mitHinweisen({ provisional: true, obscured: true })
    const { container } = renderPage(<Ranking />)

    await screen.findByText(/weniger als fünf Seiten/)
    const results = await axe(container, {
      runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'] },
    })
    expect(results.violations.map((violation) => violation.id)).toEqual([])
  })
})
