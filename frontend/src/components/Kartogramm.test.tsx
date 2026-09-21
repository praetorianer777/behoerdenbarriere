import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { Kartogramm } from './Kartogramm'
import { laender } from '../laender'

describe('Kartogramm', () => {
  it('kennt alle sechzehn Länder, jedes auf einem eigenen Platz', () => {
    expect(laender).toHaveLength(16)
    const plaetze = new Set(laender.map((land) => `${land.col}/${land.row}`))
    expect(plaetze.size).toBe(16)
  })

  it('zeigt je Land den gelieferten Text', () => {
    const werte = new Map([['Bayern', 42]])
    const { container } = render(
      <Kartogramm
        werte={werte}
        text={(wert) => (wert === undefined ? '–' : String(wert))}
        klasse={() => ''}
      />,
    )
    expect(container.textContent).toContain('BY42')
    expect(container.textContent).toContain('SH–')
  })

  // Die Kacheln sind Schmuck neben einer Tabelle. Wer sie vorgelesen bekäme, hörte
  // sechzehn Kürzel ohne Zusammenhang.
  it('ist für Hilfsmittel unsichtbar', () => {
    const { container } = render(<Kartogramm werte={new Map()} text={() => ''} klasse={() => ''} />)
    expect(container.firstElementChild).toHaveAttribute('aria-hidden', 'true')
  })
})
