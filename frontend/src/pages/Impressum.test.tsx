import { screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { Impressum } from './Impressum'
import { api } from '../api/client'
import { betreiber } from '../test/fixtures'
import { renderPage } from '../test/render'

/**
 * Die Angaben kommen vom Server, der sie aus seiner Umgebung liest — wer die Images
 * nutzt, soll für sein Impressum den Code nicht anfassen müssen.
 */
describe('Impressum', () => {
  beforeEach(() => {
    vi.spyOn(api, 'betreiber').mockResolvedValue(betreiber)
  })
  afterEach(() => vi.restoreAllMocks())

  it('zeigt die Angaben des Betreibers', async () => {
    renderPage(<Impressum />)
    // Der Name steht zweimal: in der Anschrift und beim Verantwortlichen für den Inhalt.
    expect((await screen.findAllByText(betreiber.name, { exact: false })).length).toBeGreaterThan(0)
    expect(screen.getByRole('link', { name: betreiber.email })).toHaveAttribute(
      'href',
      `mailto:${betreiber.email}`,
    )
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  // Ein erfundenes Impressum wäre schlimmer als ein fehlendes: Fehlt etwas, steht das
  // sichtbar da, und der Hinweis sagt, wo es gesetzt wird.
  it('warnt, solange Angaben fehlen', async () => {
    vi.spyOn(api, 'betreiber').mockResolvedValue({ ...betreiber, street: '', complete: false })
    renderPage(<Impressum />)
    const warnung = await screen.findByRole('alert')
    expect(warnung).toHaveTextContent(/OPERATOR_STREET/)
    expect(warnung).toHaveTextContent(/\.env/)
  })

  it('lässt Telefon und USt-IdNr. weg, wenn sie fehlen', async () => {
    renderPage(<Impressum />)
    await screen.findAllByText(betreiber.name, { exact: false })
    expect(screen.queryByText(/Telefon/)).not.toBeInTheDocument()
  })
})
