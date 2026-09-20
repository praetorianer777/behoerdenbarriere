import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { RuleList } from './RuleList'
import type { Rule } from '../api/types'

const regel = (over: Partial<Rule>): Rule => ({
  rule_id: 'color-contrast',
  impact: 'serious',
  principle: 'perceivable',
  help: 'Elements must meet minimum color contrast ratio thresholds',
  help_url: 'https://dequeuniversity.com/rules/axe/4.10/color-contrast',
  pages: 4,
  nodes: 12,
  ...over,
})

describe('Liste der Verstöße', () => {
  it('nennt die Regel auf Deutsch und sagt, wen sie ausschließt', () => {
    render(<RuleList rules={[regel({})]} heading="Verstöße" emptyText="nichts" />)

    expect(
      screen.getByRole('heading', { name: 'Zu wenig Kontrast zwischen Text und Hintergrund' }),
    ).toBeInTheDocument()
    expect(screen.getByText(/Wer schlecht sieht/)).toBeInTheDocument()
    expect(screen.queryByText(/Elements must meet minimum color contrast/)).not.toBeInTheDocument()
  })

  // Eine Regel, für die noch keine Übersetzung existiert, darf nicht als leere Zeile
  // erscheinen — der englische Satz ist besser als nichts.
  it('zeigt eine unübersetzte Regel weiter an', () => {
    render(
      <RuleList
        rules={[regel({ rule_id: 'noch-nicht-uebersetzt', help: 'Something must be something' })]}
        heading="Verstöße"
        emptyText="nichts"
      />,
    )

    expect(screen.getByRole('heading', { name: 'Something must be something' })).toBeInTheDocument()
  })

  it('verweist weiter auf die Regel zum Nachlesen', () => {
    render(<RuleList rules={[regel({})]} heading="Verstöße" emptyText="nichts" />)
    expect(screen.getByRole('link', { name: /Regel color-contrast nachlesen/ })).toBeInTheDocument()
  })
})
