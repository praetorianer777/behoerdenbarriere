import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { Ranking } from './Ranking'
import { api } from '../api/client'
import { agencyList, stats } from '../test/fixtures'
import { renderPage } from '../test/render'

describe('Sortierung des Rankings', () => {
  beforeEach(() => {
    vi.spyOn(api, 'stats').mockResolvedValue(stats)
  })
  afterEach(() => vi.restoreAllMocks())

  async function ranking() {
    const abfrage = vi.spyOn(api, 'agencies').mockResolvedValue(agencyList)
    const user = userEvent.setup()
    renderPage(<Ranking />)
    await screen.findByRole('table')
    abfrage.mockClear()
    return { user, abfrage }
  }

  // Sortiert wird in der Datenbank. Eine Liste, die seitenweise kommt, im Browser zu
  // sortieren, ordnete die angezeigten fünfzig statt aller vierhundert — und sähe
  // dabei richtig aus.
  it('reicht die Sortierung an die API weiter', async () => {
    const { user, abfrage } = await ranking()

    await user.click(screen.getByRole('button', { name: /Veränderung/ }))

    await waitFor(() => expect(abfrage).toHaveBeenCalled())
    expect(abfrage.mock.calls.at(-1)?.[0]).toMatchObject({ sort: 'delta' })
  })

  it('beginnt bei jeder Spalte mit der Richtung, nach der gefragt wird', async () => {
    const { user, abfrage } = await ranking()

    // Beim Datum interessiert zuerst das zuletzt Geprüfte, beim Namen A vor Z.
    await user.click(screen.getByRole('button', { name: /Geprüft am/ }))
    await waitFor(() => expect(abfrage.mock.calls.at(-1)?.[0]).toMatchObject({ sort: 'scanned' }))

    await user.click(screen.getByRole('button', { name: /Behörde/ }))
    await waitFor(() => expect(abfrage.mock.calls.at(-1)?.[0]).toMatchObject({ sort: 'name' }))
  })

  it('dreht die Richtung beim zweiten Klick um', async () => {
    const { user, abfrage } = await ranking()

    await user.click(screen.getByRole('button', { name: /Ebene/ }))
    await waitFor(() => expect(abfrage.mock.calls.at(-1)?.[0]).toMatchObject({ sort: 'level' }))

    await user.click(screen.getByRole('button', { name: /Ebene/ }))
    await waitFor(() =>
      expect(abfrage.mock.calls.at(-1)?.[0]).toMatchObject({ sort: 'level_desc' }),
    )
  })

  // Der Pfeil sagt sehenden Menschen, wonach sortiert ist. aria-sort sagt es allen
  // anderen.
  it('nennt die Sortierung auch ohne den Pfeil', async () => {
    const { user } = await ranking()

    const spalte = () => screen.getByRole('columnheader', { name: /Ebene/ })
    expect(spalte()).toHaveAttribute('aria-sort', 'none')

    await user.click(screen.getByRole('button', { name: /Ebene/ }))
    await waitFor(() => expect(spalte()).toHaveAttribute('aria-sort', 'ascending'))

    await user.click(screen.getByRole('button', { name: /Ebene/ }))
    await waitFor(() => expect(spalte()).toHaveAttribute('aria-sort', 'descending'))
  })

  // Am Telefon wird die Tabelle zu Karten; ohne diese Auswahl gäbe es dort keine
  // Sortierung.
  it('lässt sich auch ohne Tabelle sortieren', async () => {
    const { user, abfrage } = await ranking()

    await user.selectOptions(screen.getByLabelText('Sortierung'), 'scanned_asc')

    await waitFor(() =>
      expect(abfrage.mock.calls.at(-1)?.[0]).toMatchObject({ sort: 'scanned_asc' }),
    )
  })
})
