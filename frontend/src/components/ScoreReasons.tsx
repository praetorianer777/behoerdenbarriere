import type { PageExplanation } from '../api/types'
import { impactLabel } from '../lib'
import { regelName } from '../regeln'

function points(value: number): string {
  return value.toLocaleString('de-DE', { maximumFractionDigits: 1 })
}

function percent(share: number): string {
  return (share * 100).toLocaleString('de-DE', { maximumFractionDigits: 0 })
}

/**
 * Warum diese Seite diesen Wert hat. Zwei Zahlen je Befund, und sie beantworten
 * verschiedene Fragen: Der Anteil sagt, wo das Gewicht liegt — alle Anteile zusammen
 * ergeben das Ganze. Die Punkte sagen, was das Beheben dieses einen Befundes
 * zurückgibt; weil die Kurve nicht geradlinig ist, bringt zwei Befunde zu beheben mehr
 * als die beiden Einzelwerte zusammen.
 */
export function ScoreReasons({ page }: { page: PageExplanation }) {
  if (page.reasons.length === 0) {
    return (
      <p className="text-slate-700">
        Auf dieser Seite wurde nichts gefunden, was maschinell prüfbar wäre. Das ist nicht dasselbe
        wie barrierefrei.
      </p>
    )
  }

  return (
    <>
      <p className="text-slate-700">
        {page.reasons.length === 1 ? 'Ein Befund' : `${page.reasons.length} Befunde`} auf{' '}
        {page.dom_nodes.toLocaleString('de-DE')} Elementen ergeben {points(page.score)} von 100
        Punkten. Diese Seite zählt{' '}
        {page.weight === 3 ? 'dreifach' : page.weight === 2 ? 'doppelt' : 'einfach'} im Wert der
        Behörde
        {page.is_entry ? ' (Startseite)' : page.priority ? ' (rechtlich besonders relevant)' : ''}.
      </p>

      <div
        className="mt-3 overflow-x-auto"
        tabIndex={0}
        role="region"
        aria-label="Begründung des Werts, waagerecht scrollbar"
      >
        <table className="w-full border-collapse bg-white text-left">
          <caption className="sr-only">
            Befunde dieser Seite mit ihrem Anteil am Abzug und dem Gewinn beim Beheben
          </caption>
          <thead>
            <tr className="border-b border-slate-300">
              <th
                scope="col"
                className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
              >
                Befund
              </th>
              <th
                scope="col"
                className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
              >
                Schwere
              </th>
              <th
                scope="col"
                className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
              >
                Elemente
              </th>
              <th
                scope="col"
                className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
              >
                Anteil
              </th>
              <th
                scope="col"
                className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
              >
                Behoben: <span className="whitespace-nowrap">+ Punkte</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {page.reasons.map((reason) => (
              <tr key={reason.rule_id} className="border-b border-slate-200">
                <th scope="row" className="px-3 py-2 font-normal break-words hyphens-auto">
                  {reason.help_url ? (
                    <a href={reason.help_url} className="underline">
                      {regelName(reason)}
                    </a>
                  ) : (
                    regelName(reason)
                  )}
                </th>
                <td className="px-3 py-2">{impactLabel[reason.impact]}</td>
                <td className="px-3 py-2">{reason.nodes}</td>
                <td className="px-3 py-2">{percent(reason.share)} %</td>
                <td className="px-3 py-2">+{points(reason.points_if_fixed)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <p className="mt-2 text-sm text-slate-600">
        Der Anteil sagt, wo das Gewicht liegt; die Punkte sagen, was das Beheben dieses einen
        Befundes zurückgibt. Mehrere zu beheben bringt mehr als die Einzelwerte zusammen.
      </p>
    </>
  )
}
