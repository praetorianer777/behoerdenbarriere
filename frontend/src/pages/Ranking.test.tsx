import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { Ranking } from './Ranking'
import { api } from '../api/client'
import { agencyList, stats } from '../test/fixtures'
import { renderPage } from '../test/render'

describe('Ranking', () => {
  beforeEach(() => {
    vi.spyOn(api, 'agencies').mockResolvedValue(agencyList)
    vi.spyOn(api, 'stats').mockResolvedValue(stats)
  })
  afterEach(() => vi.restoreAllMocks())

  it('zeigt Behörden mit Score und Note', async () => {
    renderPage(<Ranking />)

    const row = await screen.findByRole('row', { name: /Bundesregierung/ })
    expect(within(row).getByText('75,4')).toBeInTheDocument()
    // Die Note steht als Text da und nicht nur als Farbe.
    expect(within(row).getByText(/Note C/)).toBeInTheDocument()
  })

  // Eine ungeprüfte Behörde darf nicht wie eine mit null Punkten aussehen.
  it('unterscheidet ungeprüft von schlecht', async () => {
    renderPage(<Ranking />)

    const row = await screen.findByRole('row', { name: /Kiel/ })
    expect(within(row).getByText('nicht geprüft')).toBeInTheDocument()
    expect(within(row).queryByText('0')).not.toBeInTheDocument()
  })

  it('meldet die Trefferzahl in einer Live-Region', async () => {
    renderPage(<Ranking />)
    const status = await screen.findByText('2 Behörden gefunden')
    // Wer die Tabelle nicht sieht, muss hören, dass sie sich geändert hat.
    expect(status).toHaveAttribute('role', 'status')
  })

  it('übernimmt Filter in die Adresse und in die Abfrage', async () => {
    const user = userEvent.setup()
    renderPage(<Ranking />)

    await screen.findByRole('table')
    await user.selectOptions(screen.getByLabelText('Ebene'), 'kommune')

    await waitFor(() => {
      expect(api.agencies).toHaveBeenCalledWith(
        expect.objectContaining({ level: 'kommune' }),
        expect.anything(),
      )
    })
  })

  it('liest den Filter aus der Adresse', async () => {
    renderPage(<Ranking />, { route: '/?level=bund&grade=C' })

    await waitFor(() => {
      expect(api.agencies).toHaveBeenCalledWith(
        expect.objectContaining({ level: 'bund', grade: 'C' }),
        expect.anything(),
      )
    })
    expect(screen.getByLabelText('Ebene')).toHaveValue('bund')
  })

  it('zeigt einen Fehler statt einer leeren Tabelle', async () => {
    vi.spyOn(api, 'agencies').mockRejectedValue(new Error('500 Internal Server Error'))
    renderPage(<Ranking />)

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('konnte nicht geladen werden')
  })
})
