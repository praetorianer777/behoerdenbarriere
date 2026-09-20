import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

import type { TrendPoint } from '../api/types'
import { formatDate, formatScore } from '../lib'

interface Props {
  history?: TrendPoint[] | null
}

/**
 * Die Kurve ist die schnelle Lesart, die Tabelle darunter die belastbare: Screenreader
 * und Tastatur kommen an ein SVG nicht heran, an eine Tabelle schon.
 */
export function TrendChart({ history }: Props) {
  const points = history ?? []
  if (points.length < 2) {
    return (
      <p className="text-slate-700">
        Für einen Verlauf braucht es mindestens zwei Prüfungen. Bisher gibt es{' '}
        {points.length === 1 ? 'eine' : 'keine'}.
      </p>
    )
  }

  const data = points.map((point) => ({
    date: formatDate(point.at),
    score: point.score,
  }))

  return (
    <figure className="m-0">
      <div aria-hidden="true" inert className="h-64 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data} margin={{ top: 8, right: 8, bottom: 8, left: 0 }}>
            <CartesianGrid stroke="#cbd5e1" strokeDasharray="3 3" />
            <XAxis dataKey="date" stroke="#334155" tick={{ fontSize: 12 }} />
            <YAxis domain={[0, 100]} stroke="#334155" tick={{ fontSize: 12 }} />
            <Tooltip />
            <Line
              type="monotone"
              dataKey="score"
              stroke="#0f172a"
              strokeWidth={2}
              dot={{ r: 3 }}
              isAnimationActive={false}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>

      <figcaption className="sr-only">
        Verlauf des Scores über {points.length} Prüfungen. Die Werte stehen in der Tabelle darunter.
      </figcaption>

      <details className="mt-2">
        <summary className="cursor-pointer text-sm underline">Werte als Tabelle</summary>
        <table className="mt-2 w-full max-w-md border-collapse bg-white text-left">
          <caption className="sr-only">Score je Prüfung</caption>
          <thead>
            <tr className="border-b border-slate-300">
              <th scope="col" className="px-3 py-2">
                Geprüft am
              </th>
              <th scope="col" className="px-3 py-2">
                Score
              </th>
              <th scope="col" className="px-3 py-2">
                Note
              </th>
            </tr>
          </thead>
          <tbody>
            {points.map((point) => (
              <tr key={point.at} className="border-b border-slate-200">
                <th scope="row" className="px-3 py-2 font-normal">
                  {formatDate(point.at)}
                </th>
                <td className="px-3 py-2">{formatScore(point.score)}</td>
                <td className="px-3 py-2">{point.grade}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </details>
    </figure>
  )
}
