import type { ContactGroup, ContactPhase, Direction, Impact } from './api/types'

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

// Die Pflichtangaben aus § 12b BGG in der Reihenfolge, in der das Gesetz sie nennt.
export const requirementLabel: Record<string, string> = {
  reachable: 'Von der Startseite aus erreichbar',
  conformance: 'Angabe, wie weit die Seite vereinbar ist',
  shortcomings: 'Auflistung der nicht barrierefreien Inhalte',
  date: 'Datum der Erstellung oder Überprüfung',
  feedback: 'Möglichkeit, Barrieren zu melden',
  enforcement: 'Hinweis auf das Schlichtungsverfahren',
}

// Die Namen, unter denen die Dienste bekannt sind. Die Zuordnung sagt, zu wem ein
// Hostname gehört — nicht, was dort mit den Daten geschieht.
export const groupLabel: Record<ContactGroup, string> = {
  'google-fonts': 'Google Fonts',
  'google-analytics': 'Google Analytics / Tag Manager',
  'google-maps': 'Google Maps',
  'google-ads': 'Google Werbung',
  'google-other': 'Google (sonstiges)',
  youtube: 'YouTube',
  vimeo: 'Vimeo',
  meta: 'Meta (Facebook, Instagram)',
  x: 'X (Twitter)',
  linkedin: 'LinkedIn',
  matomo: 'Matomo',
  etracker: 'etracker',
  'consent-tool': 'Einwilligungsdienst',
  cdn: 'Content Delivery Network',
  unknown: 'nicht zugeordnet',
}

// Der Zeitpunkt ist der eigentliche Befund: Vor der Einwilligung hatte niemand die
// Gelegenheit zu widersprechen.
export const phaseLabel: Record<ContactPhase, string> = {
  before_consent: 'vor der Einwilligung',
  after_declined: 'nach Ablehnung',
  after_accepted: 'nach Zustimmung',
}

export const phaseExplanation: Record<ContactPhase, string> = {
  before_consent:
    'Diese Hosts wurden beim Aufruf der Seite kontaktiert — bevor ein Einwilligungsbanner beantwortet werden konnte oder ohne dass eines erschien.',
  after_declined:
    'Diese Hosts wurden erst kontaktiert, nachdem wir alles Ablehnbare abgelehnt hatten.',
  after_accepted:
    'Auf diesen Seiten ließ sich nichts ablehnen, deshalb haben wir zugestimmt. Was danach geladen wurde, steht hier.',
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
