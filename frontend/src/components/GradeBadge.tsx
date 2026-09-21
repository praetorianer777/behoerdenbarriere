import { formatScore, gradeClass } from '../lib'

interface Props {
  score: number | null
  grade?: string
  size?: 'sm' | 'lg'
  /** Der Wert stützt sich auf zu wenige Seiten, um die Website zu beschreiben. */
  provisional?: boolean
  /** Gemessen wurde überwiegend die Einwilligungsabfrage, nicht die Seite dahinter. */
  obscured?: boolean
}

/**
 * Die Einschränkung hängt am Wert und nicht in einer Fußnote: Wer die Zahl zitiert,
 * soll sie mitnehmen müssen. Das kurze Wort steht sichtbar, der ganze Satz für
 * Screenreader — im Ranking ist für mehr kein Platz.
 */
function Einschraenkung({ kurz, lang }: { kurz: string; lang: string }) {
  return (
    <span className="ml-2 rounded-md border border-slate-400 px-1.5 py-0.5 align-middle text-xs font-normal text-slate-700">
      {kurz}
      <span className="sr-only"> — {lang}</span>
    </span>
  )
}

/**
 * Note und Punktzahl zusammen. Die Farbe wiederholt nur, was der Buchstabe schon
 * sagt — wer sie nicht sieht, verliert nichts.
 */
export function GradeBadge({
  score,
  grade,
  size = 'sm',
  provisional = false,
  obscured = false,
}: Props) {
  if (score === null || !grade) {
    return (
      <span className="inline-flex items-center gap-2 text-slate-600">
        <span aria-hidden="true">–</span>
        <span>nicht geprüft</span>
      </span>
    )
  }

  const big = size === 'lg'
  return (
    <span className="inline-flex items-center gap-2">
      <span
        className={`${gradeClass[grade] ?? 'bg-slate-600'} ${
          big ? 'size-16 text-4xl' : 'size-9 text-lg'
        } inline-flex items-center justify-center rounded-lg font-bold text-white`}
        aria-hidden="true"
      >
        {grade}
      </span>
      <span className={`tabular ${big ? 'text-3xl font-semibold' : 'font-semibold'}`}>
        <span className="sr-only">Note {grade}, </span>
        {formatScore(score)}
        <span className="sr-only"> von 100 Punkten</span>
        {provisional && <Einschraenkung kurz="vorläufig" lang="gestützt auf zu wenige Seiten" />}
        {obscured && (
          <Einschraenkung
            kurz="verdeckt"
            lang="überwiegend die Einwilligungsabfrage geprüft, nicht die Seite dahinter"
          />
        )}
      </span>
    </span>
  )
}
