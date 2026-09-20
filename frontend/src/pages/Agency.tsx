import { Suspense, lazy } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router'

import { api } from '../api/client'
import type { Subscores } from '../api/types'
import { DeltaBadge } from '../components/DeltaBadge'
import { GradeBadge } from '../components/GradeBadge'
import { Loading, LoadError } from '../components/Loading'
import { RuleList } from '../components/RuleList'
import {
  directionLabel,
  formatDate,
  formatMonths,
  formatScore,
  levelLabel,
  principleLabel,
} from '../lib'

// Die Diagrammbibliothek ist der größte Brocken im Bündel und wird nur auf dieser
// Seite gebraucht; nachladen spart dem Ranking zwei Drittel der Übertragung.
const TrendChart = lazy(() =>
  import('../components/TrendChart').then((module) => ({
    default: module.TrendChart,
  })),
)

export function AgencyPage() {
  const { slug = '' } = useParams()

  const agency = useQuery({
    queryKey: ['agency', slug],
    queryFn: ({ signal }) => api.agency(slug, signal),
  })
  const scan = useQuery({
    queryKey: ['scan', slug],
    queryFn: ({ signal }) => api.latestScan(slug, signal),
    // Ohne Prüfung gibt es keinen Scan; das ist kein Fehler, der wiederholt werden muss.
    retry: false,
    enabled: Boolean(agency.data?.latest_scan_id),
  })

  if (agency.isPending) return <Loading what="Die Behörde" />
  if (agency.isError) return <LoadError what="Die Behörde" error={agency.error} />

  const detail = agency.data
  const trend = detail.trend

  return (
    <>
      <p className="text-sm">
        <Link to="/" className="underline">
          Ranking
        </Link>{' '}
        <span aria-hidden="true">›</span> {detail.name}
      </p>

      <h1 className="mt-2 text-2xl font-bold break-words hyphens-auto sm:text-3xl">
        {detail.name}
      </h1>
      <p className="mt-1 text-slate-700">
        {levelLabel[detail.level]}
        {detail.state ? ` · ${detail.state}` : ''} ·{' '}
        <a href={detail.url} className="underline">
          {new URL(detail.url).host}
        </a>
      </p>

      <div className="mt-6 flex flex-wrap items-center gap-6 rounded-lg border border-slate-200 bg-white p-6">
        <GradeBadge score={detail.score} grade={detail.grade} size="lg" />
        <DeltaBadge delta={trend.delta_last} />
        <p className="text-sm text-slate-700">
          {detail.scanned_at
            ? `zuletzt geprüft am ${formatDate(detail.scanned_at)}, ${detail.pages} Seiten`
            : 'noch nicht geprüft'}
        </p>
      </div>

      {detail.score !== null && (
        <>
          <h2 className="mt-10 text-xl font-semibold">Nach WCAG-Prinzip</h2>
          <ul className="mt-3 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {(Object.keys(detail.subscores) as (keyof Subscores)[]).map((key) => (
              <li key={key} className="rounded-lg border border-slate-200 bg-white p-4">
                <h3 className="text-sm font-medium text-slate-700">{principleLabel[key]}</h3>
                <p className="mt-1 text-2xl font-semibold">{formatScore(detail.subscores[key])}</p>
              </li>
            ))}
          </ul>
        </>
      )}

      <h2 className="mt-10 text-xl font-semibold">Verlauf</h2>
      <p className="mt-2 text-slate-700">
        {trend.scans === 0
          ? 'Diese Behörde wurde noch nicht geprüft.'
          : `Bisher ${trend.scans} ${trend.scans === 1 ? 'Prüfung' : 'Prüfungen'}, zuletzt ${directionLabel[trend.direction]}.`}
        {trend.points_per_month !== undefined &&
          ` Im Schnitt ${trend.points_per_month > 0 ? '+' : ''}${trend.points_per_month} Punkte pro Monat.`}
        {trend.months_to_grade_a !== undefined &&
          ` In diesem Tempo wäre Note A in etwa ${formatMonths(trend.months_to_grade_a)} erreicht.`}
      </p>
      <div className="mt-4">
        <Suspense fallback={<p role="status">Verlauf wird geladen …</p>}>
          <TrendChart history={detail.history} />
        </Suspense>
      </div>

      {scan.data && (
        <>
          <h2 className="mt-10 text-xl font-semibold">Gefundene Barrieren</h2>
          <p className="mt-2 text-slate-700">
            Prüfung vom {formatDate(scan.data.finished_at)}: {scan.data.pages_scanned} Seiten
            geprüft
            {scan.data.pages_failed > 0 && `, ${scan.data.pages_failed} nicht erreichbar`}.
          </p>

          {scan.data.pages_blocked > 0 && (
            /* Ein Banner, das sich nicht wegklicken lässt, macht den Befund zu einem
               Befund über das Banner. Wer den Score liest, muss das wissen. */
            <p className="mt-2 rounded-lg border border-grade-d bg-white p-4">
              Auf {scan.data.pages_blocked}{' '}
              {scan.data.pages_blocked === 1 ? 'Seite ließ sich' : 'Seiten ließen sich'} die
              Einwilligungsabfrage nicht schließen. Dort beschreibt das Ergebnis das Banner und
              nicht die Seite dahinter.
            </p>
          )}

          <div className="mt-4 space-y-8">
            <RuleList
              rules={scan.data.rules ?? []}
              heading="Verstöße nach Regel"
              emptyText="Die automatisierte Prüfung hat keine Verstöße gefunden. Das heißt nicht, dass die Seite barrierefrei ist — nur, dass die maschinell prüfbaren Fehler fehlen."
            />

            {scan.data.changes && (
              <>
                <RuleList
                  rules={scan.data.changes.fixed}
                  heading="Seit der letzten Prüfung behoben"
                  emptyText="Nichts, was vorher gefunden wurde, ist verschwunden."
                />
                <RuleList
                  rules={scan.data.changes.introduced}
                  heading="Seit der letzten Prüfung neu"
                  emptyText="Keine neuen Regelverstöße."
                />
              </>
            )}
          </div>

          <h2 className="mt-10 text-xl font-semibold">Geprüfte Seiten</h2>

          {/* Auf dem Telefon als Liste: eine Adresse ist lang, und quer zu scrollen
              verdeckt genau die Spalte mit dem Score. */}
          <ul className="mt-3 space-y-3 sm:hidden">
            {(scan.data.pages ?? []).map((page) => (
              <li key={page.url} className="rounded-lg border border-slate-200 bg-white p-4">
                <a href={page.url} className="block break-words py-1 underline">
                  {page.title || page.url}
                </a>
                <p className="mt-1 text-sm break-all text-slate-600">{page.url}</p>
                <p className="mt-2 text-sm">
                  Score {formatScore(page.score)} · {page.violations}{' '}
                  {page.violations === 1 ? 'Verstoß' : 'Verstöße'}
                  {page.is_entry && ' · Startseite'}
                  {page.error && ' · nicht erreichbar'}
                  {page.consent === 'blocked' && ' · hinter Einwilligungsabfrage'}
                </p>
              </li>
            ))}
          </ul>

          <div
            className="mt-3 hidden overflow-x-auto sm:block"
            tabIndex={0}
            role="region"
            aria-label="Geprüfte Seiten, waagerecht scrollbar"
          >
            <table className="w-full border-collapse bg-white text-left">
              <caption className="sr-only">Die geprüften Seiten mit ihrem Score</caption>
              <thead>
                <tr className="border-b border-slate-300">
                  <th scope="col" className="px-3 py-2">
                    Adresse
                  </th>
                  <th scope="col" className="px-3 py-2">
                    Score
                  </th>
                  <th scope="col" className="px-3 py-2">
                    Verstöße
                  </th>
                </tr>
              </thead>
              <tbody>
                {(scan.data.pages ?? []).map((page) => (
                  <tr key={page.url} className="border-b border-slate-200">
                    <th scope="row" className="max-w-md truncate px-3 py-2 font-normal">
                      <a href={page.url} className="underline">
                        {page.title || page.url}
                      </a>
                      {page.is_entry && (
                        <span className="ml-2 text-sm text-slate-600">Startseite</span>
                      )}
                      {page.error && (
                        <span className="ml-2 text-sm text-grade-f">nicht erreichbar</span>
                      )}
                      {page.consent === 'blocked' && (
                        <span className="ml-2 text-sm text-grade-d">
                          hinter Einwilligungsabfrage
                        </span>
                      )}
                    </th>
                    <td className="px-3 py-2">{formatScore(page.score)}</td>
                    <td className="px-3 py-2">{page.violations}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </>
  )
}
