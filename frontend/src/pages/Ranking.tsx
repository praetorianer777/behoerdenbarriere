import { useEffect, useState } from 'react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router'

import { api } from '../api/client'
import { GradeBadge } from '../components/GradeBadge'
import { DeltaBadge } from '../components/DeltaBadge'
import { Loading, LoadError } from '../components/Loading'
import { formatDate, levelLabel } from '../lib'

// Lange genug, dass ein getipptes Wort eine Abfrage wird, kurz genug, dass niemand auf
// die Liste wartet.
const sucheVerzoegerung = 300

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
    // Beim Wechsel eines Filters bleibt die bisherige Liste stehen, bis die neue da
    // ist. Sonst verschwindet die Tabelle bei jedem Tastendruck und die Seite springt
    // — für Lesende mit Vergrößerung ist das nicht nur unruhig, sondern unbrauchbar.
    placeholderData: keepPreviousData,
  })
  const stats = useQuery({
    queryKey: ['stats'],
    queryFn: ({ signal }) => api.stats(signal),
  })

  // Der Suchtext wird erst getippt und dann gefragt. Jeder Tastendruck war eine
  // Abfrage über alle Behörden: „Aachen" sind sechs davon, fünf für Vorsilben, die
  // niemand gesucht hat.
  const [suche, setSuche] = useState(query.q)
  useEffect(() => {
    if (suche === query.q) return
    const timer = setTimeout(() => update({ q: suche }), sucheVerzoegerung)
    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [suche, query.q])

  // Zurück-Knopf und geteilte Adressen: Ändert sich die Adresse von außen, folgt das
  // Eingabefeld.
  useEffect(() => setSuche(params.get('q') ?? ''), [params])

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
      <h1 className="text-2xl font-bold break-words hyphens-auto sm:text-3xl">
        Wie barrierefrei sind deutsche Behörden?
      </h1>
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
            value={suche}
            onChange={(event) => setSuche(event.target.value)}
            className="mt-1 min-h-11 w-full rounded-md border border-slate-400 px-3 py-2 text-base"
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
            className="mt-1 min-h-11 w-full rounded-md border border-slate-400 px-3 py-2 text-base"
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
            className="mt-1 min-h-11 w-full rounded-md border border-slate-400 px-3 py-2 text-base"
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
            className="mt-1 min-h-11 w-full rounded-md border border-slate-400 px-3 py-2 text-base"
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

      {agencies.isPending && !agencies.isPlaceholderData ? (
        <Loading what="Das Ranking" />
      ) : agencies.isError ? (
        <LoadError what="Das Ranking" error={agencies.error} />
      ) : (
        <>
          {/* aria-busy sagt Hilfsmitteln, dass hier gerade nachgeladen wird —
              sichtbar ist es an der blasseren Liste. */}
          <div aria-busy={agencies.isFetching} className={agencies.isFetching ? 'opacity-60' : ''}>
            <p role="status" className="mt-6 text-sm text-slate-700">
              {agencies.data.total} Behörden gefunden
            </p>

            {/* Auf dem Telefon wird aus jeder Zeile eine Karte. Eine Rangliste quer
              zu scrollen hieße, die Hälfte der Behörden nicht zu sehen; WCAG 1.4.10
              verlangt außerdem, dass bei 320 px nicht in zwei Richtungen gescrollt
              werden muss. */}
            <ul className="mt-2 space-y-3 sm:hidden">
              {agencies.data.items.map((agency) => (
                <li key={agency.slug} className="rounded-lg border border-slate-200 bg-white p-4">
                  <h2 className="text-lg font-medium break-words hyphens-auto">
                    <Link to={`/behoerde/${agency.slug}`} className="block py-1 underline">
                      {agency.name}
                    </Link>
                  </h2>
                  <p className="text-sm text-slate-600">
                    {levelLabel[agency.level]}
                    {agency.state ? ` · ${agency.state}` : ''}
                  </p>
                  <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2">
                    <GradeBadge score={agency.score} grade={agency.grade} />
                    <DeltaBadge delta={agency.delta} />
                  </div>
                  <p className="mt-2 text-sm text-slate-700">
                    {agency.scanned_at
                      ? `geprüft am ${formatDate(agency.scanned_at)}`
                      : 'noch nie geprüft'}
                  </p>
                </li>
              ))}
            </ul>

            <div
              className="mt-2 hidden overflow-x-auto sm:block"
              tabIndex={0}
              role="region"
              aria-label="Rangliste, waagerecht scrollbar"
            >
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
                        onClick={() =>
                          update({
                            sort: query.sort === 'name' ? 'name_desc' : 'name',
                          })
                        }
                      />
                    </th>
                    <th scope="col" className="px-3 py-2">
                      Ebene
                    </th>
                    <th scope="col" className="px-3 py-2">
                      <SortButton
                        label="Score"
                        active={query.sort === '' || query.sort === 'score_asc'}
                        onClick={() =>
                          update({
                            sort: query.sort === 'score_asc' ? '' : 'score_asc',
                          })
                        }
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
              <nav aria-label="Seiten" className="mt-6 flex flex-wrap items-center gap-4">
                <button
                  type="button"
                  className="min-h-11 rounded-md border border-slate-400 px-4 py-2 disabled:opacity-50"
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
                  className="min-h-11 rounded-md border border-slate-400 px-4 py-2 disabled:opacity-50"
                  disabled={query.page >= pages}
                  onClick={() => update({ page: String(query.page + 1) })}
                >
                  Weiter
                </button>
              </nav>
            )}
          </div>
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
    <button type="button" onClick={onClick} className="-mx-2 min-h-11 px-2 font-semibold underline">
      {label}
      {active && <span aria-hidden="true"> ↕</span>}
    </button>
  )
}
