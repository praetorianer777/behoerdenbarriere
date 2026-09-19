import { formatDelta } from '../lib'

interface Props {
  delta?: number
}

/**
 * Veränderung zum vorigen Scan. Ohne Vorscan wird nichts behauptet — deshalb steht
 * hier nichts statt einer Null.
 */
export function DeltaBadge({ delta }: Props) {
  if (delta === undefined) return null

  const better = delta > 0
  const worse = delta < 0
  const className = better ? 'text-grade-a' : worse ? 'text-grade-f' : 'text-slate-600'
  const wording = better ? 'verbessert' : worse ? 'verschlechtert' : 'unverändert'

  return (
    <span className={`inline-flex items-center gap-1 text-sm font-medium ${className}`}>
      <span aria-hidden="true">{better ? '▲' : worse ? '▼' : '='}</span>
      <span>
        {formatDelta(delta)}
        <span className="sr-only"> Punkte gegenüber dem vorigen Scan, {wording}</span>
      </span>
    </span>
  )
}
