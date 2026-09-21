import { Link } from 'react-router'

import type { Stats } from '../api/types'
import { formatScore } from '../lib'
import { haeufigsteNote } from '../noten'

/**
 * Das ganze Bild in einer Zeile über dem Ranking. Wer nie auf „Überblick" klickt — die
 * meisten —, erfuhr sonst nicht, dass die häufigste Note ein F ist, oder dass drei
 * Viertel der Behörden noch gar nicht geprüft sind.
 *
 * Vier Zahlen, mehr nicht: Auf dem Telefon darf das Band das Ranking nicht wieder unter
 * die Falz schieben, aus der #100 es geholt hat.
 */
export function Kennzahlen({ stats }: { stats: Stats | undefined }) {
  if (!stats || !stats.grades) return null
  const spitze = haeufigsteNote(stats.grades)

  return (
    <section aria-label="Das Bild in Zahlen" className="mt-6">
      <dl className="grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-slate-200 bg-slate-200 sm:grid-cols-4">
        <Zahl name="Behörden erfasst" wert={String(stats.agencies)} />
        <Zahl name="davon geprüft" wert={String(stats.scanned)} />
        <Zahl name="Durchschnitt der geprüften" wert={formatScore(stats.avg_score)} />
        <Zahl name="häufigste Note" wert={spitze ?? '–'} />
      </dl>
      <p className="mt-2 text-sm">
        <Link to="/dashboard" className="underline">
          Zum Überblick
        </Link>
      </p>
    </section>
  )
}

function Zahl({ name, wert }: { name: string; wert: string }) {
  return (
    <div className="bg-white px-4 py-3">
      <dt className="text-xs font-semibold tracking-wide text-slate-600 uppercase">{name}</dt>
      <dd className="tabular mt-0.5 text-2xl font-bold tracking-tight">{wert}</dd>
    </div>
  )
}
