import { gradeClass } from '../lib'
import { noten, notenSatz } from '../noten'

/**
 * Sechs Balken, an jedem die Zahl. Der stärkste Befund des ganzen Projekts stand
 * vorher als Tabellenzeile mit einem sechs Pixel hohen Strich daneben.
 *
 * Kein Diagramm-Paket: Sechs Breiten in Prozent braucht keines, und ein SVG könnte
 * ein Screenreader nicht lesen. Die Liste liest sich als „F, 32 Behörden" — der Balken
 * ist Schmuck und als solcher markiert.
 */
export function Notenverteilung({
  grades,
  scanned,
}: {
  grades: Record<string, number>
  scanned: number
}) {
  const meiste = Math.max(1, ...noten.map((note) => grades[note] ?? 0))

  return (
    <figure className="m-0">
      <p className="text-lg font-medium">{notenSatz(grades, scanned)}</p>
      <ul className="mt-4 space-y-2">
        {noten.map((note) => {
          const zahl = grades[note] ?? 0
          return (
            <li key={note} className="flex items-center gap-3">
              <span
                className={`${gradeClass[note]} inline-flex size-8 shrink-0 items-center justify-center rounded-md font-bold text-white`}
              >
                {note}
              </span>
              <span className="relative h-8 grow rounded-md bg-slate-100">
                <span
                  aria-hidden="true"
                  className={`${gradeClass[note]} block h-full rounded-md`}
                  style={{ width: `${(zahl / meiste) * 100}%`, minWidth: zahl ? '4px' : 0 }}
                />
              </span>
              <span className="tabular w-14 text-right text-lg font-semibold">
                {zahl}
                <span className="sr-only"> {zahl === 1 ? 'Behörde' : 'Behörden'}</span>
              </span>
            </li>
          )
        })}
      </ul>
    </figure>
  )
}
