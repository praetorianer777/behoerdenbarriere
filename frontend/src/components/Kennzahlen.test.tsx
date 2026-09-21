import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'

import { Kennzahlen } from './Kennzahlen'
import { stats } from '../test/fixtures'

function zeige(element: React.ReactElement) {
  return render(<MemoryRouter>{element}</MemoryRouter>)
}

describe('Kennzahlen über dem Ranking', () => {
  it('nennt erfasst, geprüft, Durchschnitt und die häufigste Note', () => {
    zeige(<Kennzahlen stats={{ ...stats, grades: { A: 1, F: 3 } }} />)

    const werte = screen.getAllByRole('definition').map((dd) => dd.textContent)
    expect(werte).toEqual([String(stats.agencies), String(stats.scanned), '68,5', 'F'])
  })

  it('verweist auf den Überblick', () => {
    zeige(<Kennzahlen stats={stats} />)
    expect(screen.getByRole('link', { name: 'Zum Überblick' })).toHaveAttribute(
      'href',
      '/dashboard',
    )
  })

  // Die Statistik lädt getrennt vom Ranking. Solange sie fehlt, fehlt das Band — kein
  // leerer Rahmen, keine Nullen.
  it('zeigt ohne Statistik nichts', () => {
    const { container } = zeige(<Kennzahlen stats={undefined} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('erfindet bei Gleichstand keine häufigste Note', () => {
    zeige(<Kennzahlen stats={{ ...stats, grades: { A: 2, F: 2 } }} />)
    const werte = screen.getAllByRole('definition').map((dd) => dd.textContent)
    expect(werte[3]).toBe('A und F')
  })
})
