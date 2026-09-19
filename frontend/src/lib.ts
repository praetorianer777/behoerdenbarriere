import type { Direction, Impact } from './api/types'

// Die Notenfarben sind so gewählt, dass sie auf Weiß mindestens 4.5:1 erreichen —
// und die Note steht immer als Buchstabe daneben, damit sie nicht allein die Farbe
// tragen muss.
export const gradeClass: Record<string, string> = {
  A: 'bg-grade-a',
  B: 'bg-grade-b',
  C: 'bg-grade-c',
  D: 'bg-grade-d',
  E: 'bg-grade-e',
  F: 'bg-grade-f',
}

export const impactLabel: Record<Impact, string> = {
  critical: 'kritisch',
  serious: 'schwer',
  moderate: 'mittel',
  minor: 'gering',
}

export const levelLabel: Record<string, string> = {
  bund: 'Bund',
  land: 'Land',
  kreis: 'Kreis',
  kommune: 'Kommune',
}

export const principleLabel: Record<string, string> = {
  perceivable: 'Wahrnehmbar',
  operable: 'Bedienbar',
  understandable: 'Verständlich',
  robust: 'Robust',
}

export const directionLabel: Record<Direction, string> = {
  improved: 'verbessert',
  declined: 'verschlechtert',
  unchanged: 'unverändert',
  unknown: 'noch kein Vergleich',
}

export function formatScore(score: number | null | undefined): string {
  if (score === null || score === undefined) return 'nicht geprüft'
  return score.toLocaleString('de-DE', { maximumFractionDigits: 1 })
}

export function formatDelta(delta: number | null | undefined): string {
  if (delta === null || delta === undefined) return ''
  const value = Math.abs(delta).toLocaleString('de-DE', { maximumFractionDigits: 1 })
  if (delta > 0) return `+${value}`
  if (delta < 0) return `−${value}`
  return '±0'
}

export function formatDate(value: string | undefined | null): string {
  if (!value) return ''
  return new Date(value).toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

export function formatMonths(months: number | undefined): string {
  if (months === undefined) return ''
  if (months < 1) return 'weniger als einen Monat'
  if (months < 24) return `${Math.round(months)} Monate`
  return `${(months / 12).toLocaleString('de-DE', { maximumFractionDigits: 1 })} Jahre`
}
