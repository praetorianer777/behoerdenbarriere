import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { ThirdParties } from './ThirdParties'
import { latestScan } from '../test/fixtures'

const contacts = latestScan.third_parties!

describe('Drittanbieter einer Behörde', () => {
  it('nennt zuerst, was vor der Einwilligung geladen wurde', () => {
    render(<ThirdParties contacts={contacts} />)

    expect(screen.getByText(/1 fremder Host/)).toBeInTheDocument()
    expect(screen.getByText('fonts.gstatic.com')).toBeInTheDocument()
  })

  it('zählt öffentliche Stellen nicht als Datenabfluss, zeigt sie aber', () => {
    render(<ThirdParties contacts={contacts} />)

    // Zwei Hosts vor der Einwilligung, einer davon eine öffentliche Stelle: gewarnt
    // wird vor einem.
    expect(screen.queryByText(/2 fremde Hosts/)).not.toBeInTheDocument()
    expect(screen.getByText('www.service.bund.de')).toBeInTheDocument()
    expect(screen.getByText(/öffentliche Stelle/)).toBeInTheDocument()
  })

  it('hält die Phasen auseinander', () => {
    render(<ThirdParties contacts={contacts} />)

    expect(screen.getByRole('heading', { name: 'vor der Einwilligung' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'nach Zustimmung' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'nach Ablehnung' })).not.toBeInTheDocument()
  })

  it('sagt, dass ein Hostname kein Urteil ist', () => {
    render(<ThirdParties contacts={contacts} />)
    expect(screen.getByText(/nicht der Empfänger der Daten/)).toBeInTheDocument()
  })

  it('zeigt nichts, wenn nichts beobachtet wurde', () => {
    const { container } = render(<ThirdParties contacts={[]} />)
    expect(container).toBeEmptyDOMElement()
  })
})
