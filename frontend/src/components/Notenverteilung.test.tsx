import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { Notenverteilung } from './Notenverteilung'
import { haeufigsteNote, notenSatz } from '../noten'

describe('Satz über der Notenverteilung', () => {
  it('nennt die häufigste Note mit Zahl und Anteil', () => {
    expect(notenSatz({ A: 26, B: 6, C: 11, D: 10, E: 6, F: 32 }, 91)).toBe(
      'Die häufigste Note ist F: 32 von 91 geprüften Behörden, 35 Prozent.',
    )
  })

  // Ein geschriebener Satz bliebe stehen, wenn die Zahlen ihn längst widerlegen.
  it('wechselt mit den Zahlen', () => {
    expect(notenSatz({ A: 40, F: 3 }, 43)).toMatch(/häufigste Note ist A/)
  })

  it('nennt bei Gleichstand beide', () => {
    expect(notenSatz({ A: 5, F: 5 }, 10)).toBe(
      'Die häufigsten Noten sind A und F, je 5 von 10 geprüften Behörden.',
    )
  })

  it('erfindet ohne Prüfungen keine Note', () => {
    expect(notenSatz({}, 0)).toBe('Noch keine Behörde ist geprüft.')
  })
})

describe('Notenverteilung', () => {
  it('liest sich ohne Balken als Note und Zahl', () => {
    render(<Notenverteilung grades={{ A: 26, F: 32 }} scanned={58} />)
    const eintraege = screen.getAllByRole('listitem').map((li) => li.textContent?.trim())
    expect(eintraege[0]).toBe('A26 Behörden')
    expect(eintraege[5]).toBe('F32 Behörden')
    expect(eintraege[1]).toBe('B0 Behörden')
  })

  it('zeichnet den längsten Balken für die häufigste Note', () => {
    const { container } = render(<Notenverteilung grades={{ A: 16, F: 32 }} scanned={48} />)
    const balken = Array.from(container.querySelectorAll('[aria-hidden="true"]')) as HTMLElement[]
    expect(balken[5].style.width).toBe('100%')
    expect(balken[0].style.width).toBe('50%')
    expect(balken[1].style.width).toBe('0%')
  })
})

describe('häufigste Note', () => {
  it('nennt eine, bei Gleichstand alle', () => {
    expect(haeufigsteNote({ A: 1, F: 3 })).toBe('F')
    expect(haeufigsteNote({ A: 2, C: 2, F: 1 })).toBe('A und C')
  })

  it('nennt ohne Prüfungen keine', () => {
    expect(haeufigsteNote({})).toBeUndefined()
  })
})
