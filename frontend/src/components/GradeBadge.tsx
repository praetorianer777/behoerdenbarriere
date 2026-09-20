import { formatScore, gradeClass } from '../lib'

interface Props {
  score: number | null
  grade?: string
  size?: 'sm' | 'lg'
}

/**
 * Note und Punktzahl zusammen. Die Farbe wiederholt nur, was der Buchstabe schon
 * sagt — wer sie nicht sieht, verliert nichts.
 */
export function GradeBadge({ score, grade, size = 'sm' }: Props) {
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
          big ? 'size-14 text-3xl' : 'size-8 text-base'
        } inline-flex items-center justify-center rounded-lg font-bold text-white`}
        aria-hidden="true"
      >
        {grade}
      </span>
      <span className={big ? 'text-2xl font-semibold' : ''}>
        <span className="sr-only">Note {grade}, </span>
        {formatScore(score)}
        <span className="sr-only"> von 100 Punkten</span>
      </span>
    </span>
  )
}
