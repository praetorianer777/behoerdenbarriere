import { useQuery } from '@tanstack/react-query'

import type { Group } from '../api/types'
import { api } from '../api/client'
import { Loading, LoadError } from '../components/Loading'
import { formatDate, formatScore, gradeClass, impactLabel, levelLabel } from '../lib'
import { regelName } from '../regeln'

const grades = ['A', 'B', 'C', 'D', 'E', 'F']

export function Dashboard() {
  const stats = useQuery({
    queryKey: ['stats'],
    queryFn: ({ signal }) => api.stats(signal),
  })

  if (stats.isPending) return <Loading what="Der Überblick" />
  if (stats.isError) return <LoadError what="Der Überblick" error={stats.error} />

  const data = stats.data
  const maxGrade = Math.max(1, ...grades.map((grade) => data.grades[grade] ?? 0))

  return (
    <>
      <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">Überblick</h1>
      <p className="mt-2 text-slate-700">
        Stand: {data.updated_at ? formatDate(data.updated_at) : 'noch keine Prüfung'}
      </p>

      <ul className="mt-6 grid gap-4 sm:grid-cols-3">
        <li className="rounded-lg border border-slate-200 bg-white p-4">
          <h2 className="text-sm font-medium text-slate-700">Behörden erfasst</h2>
          <p className="mt-1 text-3xl font-semibold">{data.agencies}</p>
        </li>
        <li className="rounded-lg border border-slate-200 bg-white p-4">
          <h2 className="text-sm font-medium text-slate-700">davon geprüft</h2>
          <p className="mt-1 text-3xl font-semibold">{data.scanned}</p>
        </li>
        <li className="rounded-lg border border-slate-200 bg-white p-4">
          <h2 className="text-sm font-medium text-slate-700">Durchschnitt der geprüften</h2>
          <p className="mt-1 text-3xl font-semibold">{formatScore(data.avg_score)}</p>
        </li>
      </ul>

      <h2 className="mt-10 text-xl font-semibold">Notenverteilung</h2>
      <div
        className="mt-3 overflow-x-auto"
        tabIndex={0}
        role="region"
        aria-label="Tabelle, waagerecht scrollbar"
      >
        <table className="w-full max-w-lg border-collapse bg-white text-left">
          <caption className="sr-only">Anzahl der Behörden je Note</caption>
          <thead>
            <tr className="border-b border-slate-300">
              <th
                scope="col"
                className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
              >
                Note
              </th>
              <th
                scope="col"
                className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
              >
                Behörden
              </th>
              <th
                scope="col"
                className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
              >
                Anteil
              </th>
            </tr>
          </thead>
          <tbody>
            {grades.map((grade) => {
              const count = data.grades[grade] ?? 0
              return (
                <tr key={grade} className="border-b border-slate-200">
                  <th scope="row" className="px-3 py-2 font-normal">
                    {grade}
                  </th>
                  <td className="px-3 py-2">{count}</td>
                  <td className="px-3 py-2">
                    {/* Der Balken bebildert nur, was die Zahl daneben schon sagt. */}
                    <span
                      aria-hidden="true"
                      className={`inline-block h-3 rounded ${gradeClass[grade]}`}
                      style={{
                        width: `${(count / maxGrade) * 100}%`,
                        minWidth: count ? '4px' : 0,
                      }}
                    />
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      <h2 className="mt-10 text-xl font-semibold">Nach Ebene</h2>
      <GroupTable
        caption="Behörden je Ebene, wie viele davon geprüft sind und ihr Durchschnitt"
        rows={data.by_level}
        labels={levelLabel}
      />

      <h2 className="mt-10 text-xl font-semibold">Nach Bundesland</h2>
      <GroupTable
        caption="Behörden je Bundesland, wie viele davon geprüft sind und ihr Durchschnitt"
        rows={data.by_state}
      />

      <h2 className="mt-10 text-xl font-semibold">Häufigste Barrieren</h2>
      <p className="mt-2 text-slate-700">
        Gezählt wird je Behörde, nicht je Element — eine Seite mit tausend Bildern entscheidet die
        Liste sonst allein.
      </p>
      <div
        className="mt-3 overflow-x-auto"
        tabIndex={0}
        role="region"
        aria-label="Tabelle, waagerecht scrollbar"
      >
        <table className="w-full border-collapse bg-white text-left">
          <caption className="sr-only">
            Die häufigsten Barrieren mit ihrer Schwere und Verbreitung
          </caption>
          <thead>
            <tr className="border-b border-slate-300">
              <th
                scope="col"
                className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
              >
                Barriere
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
                Behörden
              </th>
              <th
                scope="col"
                className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
              >
                Seiten
              </th>
            </tr>
          </thead>
          <tbody>
            {data.top_rules.map((rule) => (
              <tr key={rule.rule_id} className="border-b border-slate-200">
                <th scope="row" className="px-3 py-2 font-normal break-words hyphens-auto">
                  {regelName(rule)}
                </th>
                <td className="px-3 py-2">{impactLabel[rule.impact]}</td>
                <td className="px-3 py-2">{rule.agencies}</td>
                <td className="px-3 py-2">{rule.pages}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}

function GroupTable({
  caption,
  rows,
  labels,
}: {
  caption: string
  rows: Group[]
  labels?: Record<string, string>
}) {
  return (
    <div
      className="mt-3 overflow-x-auto"
      tabIndex={0}
      role="region"
      aria-label="Tabelle, waagerecht scrollbar"
    >
      <table className="w-full max-w-2xl border-collapse bg-white text-left">
        <caption className="sr-only">{caption}</caption>
        <thead>
          <tr className="border-b border-slate-300">
            <th
              scope="col"
              className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
            >
              Name
            </th>
            <th
              scope="col"
              className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
            >
              Behörden
            </th>
            <th
              scope="col"
              className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
            >
              davon geprüft
            </th>
            <th
              scope="col"
              className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
            >
              Durchschnitt der geprüften
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.name} className="border-b border-slate-200">
              <th scope="row" className="px-3 py-2 font-normal">
                {labels?.[row.name] ?? row.name}
              </th>
              <td className="px-3 py-2">{row.agencies}</td>
              <td className="px-3 py-2">{row.scanned}</td>
              <td className="px-3 py-2">{formatScore(row.avg_score)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
