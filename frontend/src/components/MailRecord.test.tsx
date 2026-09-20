import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { MailRecord } from './MailRecord'
import { mailRecord } from '../test/fixtures'

describe('E-Mail-Einträge einer Behörde', () => {
  it('nennt den Betreiber und den Roheintrag dazu', () => {
    render(<MailRecord mail={mailRecord} />)

    expect(screen.getByText('Microsoft 365')).toBeInTheDocument()
    // Ohne den Rohwert ist die Einordnung nicht nachprüfbar.
    expect(screen.getByText('bundesregierung-de.mail.protection.outlook.com')).toBeInTheDocument()
  })

  it('sagt beim SPF-Eintrag dazu, dass er kein Postfach bedeutet', () => {
    render(<MailRecord mail={mailRecord} />)

    expect(screen.getByRole('heading', { name: 'SPF-Eintrag' })).toBeInTheDocument()
    expect(screen.getByText(/nicht, wo die Postfächer liegen/)).toBeInTheDocument()
  })

  it('übersetzt die DMARC-Regel', () => {
    render(<MailRecord mail={mailRecord} />)
    expect(screen.getByText(/abweisen/)).toBeInTheDocument()
  })

  it('schweigt über die E-Mail, wenn die Abfrage scheiterte', () => {
    render(<MailRecord mail={{ ...mailRecord, error: 'timeout' }} />)

    expect(screen.getByText(/blieb ohne Antwort/)).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'MX-Einträge' })).not.toBeInTheDocument()
  })
})
