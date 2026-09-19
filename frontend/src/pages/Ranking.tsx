import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router'

import { api } from '../api/client'
import { GradeBadge } from '../components/GradeBadge'
import { DeltaBadge } from '../components/DeltaBadge'
import { Loading, LoadError } from '../components/Loading'
import { formatDate, levelLabel } from '../lib'

const grades = ['A', 'B', 'C', 'D', 'E', 'F']
const levels = ['bund', 'land', 'kreis', 'kommune']

export function Ranking() {
  // Die Filter stehen in der Adresse, damit ein Stand teilbar und der Zurück-Knopf
  // brauchbar bleibt.
  const [params, setParams] = useSearchParams()
  const query = {
    q: params.get('q') ?? '',
    level: params.get('level') ?? '',
    state: params.get('state') ?? '',
    grade: params.get('grade') ?? '',
    sort: params.get('sort') ?? '',
    page: Number(params.get('page') ?? 1),
    per_page: 50,
  }

  const agencies = useQuery({
    queryKey: ['agencies', query],
    queryFn: ({ signal }) => api.agencies(query, signal),
  })
  const stats = useQuery({ queryKey: ['stats'], queryFn: ({ signal }) => api.stats(signal) })

  function update(changes: Record<string, string>) {
    const next = new URLSearchParams(params)
    for (const [key, value] of Object.entries(changes)) {
      if (value) next.set(key, value)
      else next.delete(key)
    }
    // Jede Änderung an einem Filter fängt wieder auf Seite eins an.
    if (!('page' in changes)) next.delete('page')
    setParams(next)
  }

  const pages = agencies.data ? Math.ceil(agencies.data.total / agencies.data.per_page) : 0

  return (
    <>
      <h1 className="text-3xl font-bold">Wie barrierefrei sind deutsche Behörden?</h1>
      <p className="mt-2 max-w-2xl text-slate-700">
        Jede Website wird automatisiert nach WCAG 2.1 AA geprüft. Je höher der Wert, desto weniger
        Barrieren wurden gefunden.
      </p>

      <form
        className="mt-6 grid gap-4 rounded-lg border border-slate-200 bg-white p-4 sm:grid-cols-2 lg:grid-cols-4"
        onSubmit={(event) => event.preventDefault()}
      >
        <div>
          <label htmlFor="suche" className="block text-sm font-medium">
            Behörde suchen
          </label>
          <input
            id="suche"
            type="search"
            defaultValue={query.q}
            onChange={(event) => update({ q: event.target.value })}
            className="mt-1 w-full rounded-md border border-slate-400 px-3 py-2"
          />
        </div>

        <div>
          <label htmlFor="ebene" className="block text-sm font-medium">
            Ebene
          </label>
          <select
            id="ebene"
            value={query.level}
            onChange={(event) => update({ level: event.target.value })}
            className="mt-1 w-full rounded-md border border-slate-400 px-3 py-2"
          >
            <option value="">alle</option>
            {levels.map((level) => (
              <option key={level} value={level}>
                {levelLabel[level]}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label htmlFor="bundesland" className="block text-sm font-medium">
            Bundesland
          </label>
          <select
            id="bundesland"
            value={query.state}
            onChange={(event) => update({ state: event.target.value })}
            className="mt-1 w-full rounded-md border border-slate-400 px-3 py-2"
          >
            <option value="">alle</option>
            {(stats.data?.states ?? []).map((state) => (
              <option key={state} value={state}>
                {state}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label htmlFor="note" className="block text-sm font-medium">
            Note
          </label>
          <select
            id="note"
            value={query.grade}
            onChange={(event) => update({ grade: event.target.value })}
            className="mt-1 w-full rounded-md border border-slate-400 px-3 py-2"
          >
            <option value="">alle</option>
            {grades.map((grade) => (
              <option key={grade} value={grade}>
                {grade}
              </option>
            ))}
          </select>
        </div>
      </form>

      {agencies.isPending ? (
        <Loading what="Das Ranking" />
      ) : agencies.isError ? (
        <LoadError what="Das Ranking" error={agencies.error} />
      ) : (
        <>
          <p role="status" className="mt-6 text-sm text-slate-700">
            {agencies.data.total} Behörden gefunden
          </p>

          <div className="mt-2 overflow-x-auto">
            <table className="w-full border-collapse bg-white text-left">
              <caption className="sr-only">
                Behörden mit ihrem Barrierefreiheits-Score, sortierbar nach Wert und Name
              </caption>
              <thead>
                <tr className="border-b border-slate-300">
                  <th scope="col" className="px-3 py-2">
                    <SortButton
                      label="Behörde"
                      active={query.sort === 'name' || query.sort === 'name_desc'}
                      onClick={() => update({ sort: query.sort === 'name' ? 'name_desc' : 'name' })}
                    />
                  </th>
                  <th scope="col" className="px-3 py-2">
                    Ebene
                  </th>
                  <th scope="col" className="px-3 py-2">
                    <SortButton
                      label="Score"
                      active={query.sort === '' || query.sort === 'score_asc'}
                      onClick={() => update({ sort: query.sort === 'score_asc' ? '' : 'score_asc' })}
                    />
                  </th>
                  <th scope="col" className="px-3 py-2">
                    Veränderung
                  </th>
                  <th scope="col" className="px-3 py-2">
                    Geprüft am
                  </th>
                </tr>
              </thead>
              <tbody>
                {agencies.data.items.map((agency) => (
                  <tr key={agency.slug} className="border-b border-slate-200">
                    <th scope="row" className="px-3 py-3 font-medium">
                      <Link to={`/behoerde/${agency.slug}`} className="underline">
                        {agency.name}
                      </Link>
                      {agency.state && (
                        <span className="block text-sm font-normal text-slate-600">
                          {agency.state}
                        </span>
                      )}
                    </th>
                    <td className="px-3 py-3">{levelLabel[agency.level]}</td>
                    <td className="px-3 py-3">
                      <GradeBadge score={agency.score} grade={agency.grade} />
                    </td>
                    <td className="px-3 py-3">
                      <DeltaBadge delta={agency.delta} />
                    </td>
                    <td className="px-3 py-3 text-sm text-slate-700">
                      {formatDate(agency.scanned_at) || 'noch nie'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {pages > 1 && (
            <nav aria-label="Seiten" className="mt-6 flex items-center gap-4">
              <button
                type="button"
                className="rounded-md border border-slate-400 px-3 py-2 disabled:opacity-50"
                disabled={query.page <= 1}
                onClick={() => update({ page: String(query.page - 1) })}
              >
                Zurück
              </button>
              <span>
                Seite {query.page} von {pages}
              </span>
              <button
                type="button"
                className="rounded-md border border-slate-400 px-3 py-2 disabled:opacity-50"
                disabled={query.page >= pages}
                onClick={() => update({ page: String(query.page + 1) })}
              >
                Weiter
              </button>
            </nav>
          )}
        </>
      )}
    </>
  )
}

function SortButton({
  label,
  active,
  onClick,
}: {
  label: string
  active: boolean
  onClick: () => void
}) {
  return (
    <button type="button" onClick={onClick} className="font-semibold underline">
      {label}
      {active && <span aria-hidden="true"> ↕</span>}
    </button>
  )
}
