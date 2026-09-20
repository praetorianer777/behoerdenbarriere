import type { StatementResult } from '../api/types'
import { requirementLabel } from '../lib'

/**
 * Die Erklärung zur Barrierefreiheit ist keine Kennzahl, sondern eine Pflicht aus
 * § 12b BGG und § 7 BITV 2.0. Sie steht deshalb neben dem Score und nicht darin: Ein
 * fehlender Nachweis ist ein Rechtsverstoß, keine Barriere in der Seite.
 *
 * Geprüft wird nur, ob die vorgeschriebenen Angaben da sind — ob die Angaben stimmen,
 * kann kein Programm beurteilen. Deshalb steht zu jedem Punkt der Fundort dabei.
 */
export function StatementCheck({ statement }: { statement: StatementResult }) {
  const met = statement.findings.filter((finding) => finding.met).length
  const total = statement.findings.length

  return (
    <section className="mt-10">
      <h2 className="text-xl font-semibold">Erklärung zur Barrierefreiheit</h2>

      {!statement.found ? (
        <p className="mt-2 rounded-lg border border-grade-f bg-white p-4">
          Auf den geprüften Seiten wurde keine Erklärung zur Barrierefreiheit gefunden. § 12b des
          Behindertengleichstellungsgesetzes verpflichtet öffentliche Stellen dazu, eine zu
          veröffentlichen und von jeder Seite aus erreichbar zu machen.
        </p>
      ) : (
        <>
          <p className="mt-2 text-slate-700">
            {met} von {total} Pflichtangaben gefunden
            {statement.url && (
              <>
                {' '}
                in{' '}
                <a href={statement.url} className="break-all underline">
                  der Erklärung
                </a>
              </>
            )}
            .
          </p>

          <ul className="mt-3 space-y-2">
            {statement.findings.map((finding) => (
              <li
                key={finding.requirement}
                className="rounded-lg border border-slate-200 bg-white p-4"
              >
                <p className="flex items-baseline gap-2 font-medium">
                  {/* Das Zeichen wiederholt nur, was daneben steht — wer es nicht
                      sieht, verliert nichts. */}
                  <span
                    aria-hidden="true"
                    className={finding.met ? 'text-grade-a' : 'text-grade-f'}
                  >
                    {finding.met ? '✓' : '✗'}
                  </span>
                  <span>
                    <span className="sr-only">{finding.met ? 'Vorhanden: ' : 'Fehlt: '}</span>
                    {requirementLabel[finding.requirement] ?? finding.requirement}
                  </span>
                </p>
                {finding.evidence && (
                  <p className="mt-1 text-sm break-words text-slate-600 italic">
                    „{finding.evidence}“
                  </p>
                )}
              </li>
            ))}
          </ul>
        </>
      )}

      <p className="mt-3 text-sm text-slate-600">
        Geprüft wird, ob die vorgeschriebenen Angaben vorhanden sind — nicht, ob sie zutreffen. Ob
        eine Behörde ihre Website zu Recht als „teilweise vereinbar“ bezeichnet, kann nur ein Mensch
        beurteilen.
      </p>
    </section>
  )
}
