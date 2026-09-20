import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { Fehlergrenze } from './Fehlergrenze'

function Kaputt(): never {
  throw new Error('history.length ist nicht lesbar')
}

describe('Fehlergrenze', () => {
  beforeEach(() => {
    // React schreibt den Fehler selbst in die Konsole; im Testlauf ist das nur Lärm.
    vi.spyOn(console, 'error').mockImplementation(() => undefined)
  })
  afterEach(() => vi.restoreAllMocks())

  it('zeigt eine Meldung statt einer leeren Seite', () => {
    render(
      <Fehlergrenze>
        <Kaputt />
      </Fehlergrenze>,
    )

    expect(screen.getByRole('alert')).toHaveTextContent('konnte nicht aufgebaut werden')
    // Die technische Meldung gehört dazu, sonst ist der Fehler nicht weiterzugeben.
    expect(screen.getByText(/history\.length/)).toBeInTheDocument()
  })

  it('sagt, dass es an uns liegt, und bietet einen Weg zurück', () => {
    render(
      <Fehlergrenze>
        <Kaputt />
      </Fehlergrenze>,
    )

    expect(screen.getByText(/liegt an uns/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Ranking' })).toHaveAttribute('href', '/')
  })

  it('lässt heile Inhalte in Ruhe', () => {
    render(
      <Fehlergrenze>
        <p>Alles in Ordnung</p>
      </Fehlergrenze>,
    )

    expect(screen.getByText('Alles in Ordnung')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
