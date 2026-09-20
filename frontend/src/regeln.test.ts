import { describe, expect, it } from 'vitest'

import { regelName, regelWirkung, regeln } from './regeln'

describe('Regelnamen auf Deutsch', () => {
  it('übersetzt eine bekannte Regel', () => {
    expect(regelName({ rule_id: 'color-contrast', help: 'Elements must meet minimum…' })).toBe(
      'Zu wenig Kontrast zwischen Text und Hintergrund',
    )
  })

  // Eine fehlende Übersetzung darf keine leere Zeile ergeben: lieber der englische
  // Satz der API als gar nichts.
  it('fällt auf den Text der Schnittstelle zurück', () => {
    expect(regelName({ rule_id: 'gibt-es-nicht', help: 'Something must be something' })).toBe(
      'Something must be something',
    )
  })

  it('fällt zuletzt auf die Kennung zurück', () => {
    expect(regelName({ rule_id: 'gibt-es-nicht' })).toBe('gibt-es-nicht')
  })

  it('hat keine Wirkung für eine unbekannte Regel', () => {
    expect(regelWirkung('gibt-es-nicht')).toBeUndefined()
  })

  it('deckt die Regeln ab, die in den Prüfungen vorkommen', () => {
    // Die zwanzig häufigsten aus dem Bestand. Kommt eine neue dazu, fällt sie auf
    // Englisch zurück — das ist der Zustand, den dieser Test klein halten soll.
    const haeufig = [
      'color-contrast',
      'link-name',
      'aria-allowed-attr',
      'aria-hidden-focus',
      'aria-required-children',
      'list',
      'image-alt',
      'button-name',
      'meta-viewport',
      'frame-title',
      'link-in-text-block',
      'aria-required-parent',
      'html-has-lang',
      'scrollable-region-focusable',
      'listitem',
      'label',
      'nested-interactive',
      'aria-prohibited-attr',
      'aria-required-attr',
      'aria-input-field-name',
    ]
    const fehlen = haeufig.filter((id) => !(id in regeln))
    expect(fehlen).toEqual([])
  })

  it('nennt zu jeder Regel einen Namen und die betroffene Person', () => {
    for (const [id, regel] of Object.entries(regeln)) {
      expect(regel.name.length, id).toBeGreaterThan(5)
      expect(regel.wen.length, id).toBeGreaterThan(20)
      // Die Kennung durchzureichen wäre keine Übersetzung.
      expect(regel.name, id).not.toBe(id)
    }
  })
})
