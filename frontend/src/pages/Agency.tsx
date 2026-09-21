import { Suspense, lazy } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router'

import { api } from '../api/client'
import type { Subscores } from '../api/types'
import { DeltaBadge } from '../components/DeltaBadge'
import { GradeBadge } from '../components/GradeBadge'
import { Loading, LoadError } from '../components/Loading'
import { RuleList } from '../components/RuleList'
import { ScoreReasons } from '../components/ScoreReasons'
import { StatementCheck } from '../components/StatementCheck'
import { ThirdParties } from '../components/ThirdParties'
import { MailRecord } from '../components/MailRecord'
import {
  directionLabel,
  formatDate,
  formatMonths,
  formatScore,
  impactLabel,
  levelLabel,
  principleExplanation,
  principleLabel,
} from '../lib'
import { regelName, regelWirkung } from '../regeln'

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

      <h1 className="mt-2 text-3xl font-bold tracking-tight break-words hyphens-auto sm:text-4xl">
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
        <GradeBadge
          score={detail.score}
          grade={detail.grade}
          size="lg"
          provisional={detail.provisional}
          obscured={detail.obscured}
        />
        <DeltaBadge delta={trend.delta_last} />
        <p className="text-sm text-slate-700">
          {detail.scanned_at
            ? `zuletzt geprüft am ${formatDate(detail.scanned_at)}, ${detail.pages} Seiten`
            : 'noch nicht geprüft'}
        </p>
      </div>

      {detail.provisional && (
        /* Der Grund steht dazu: Sonst liest sich „vorläufig" wie ein Versäumnis der
           Behörde, dabei ist es ihre robots.txt, die wir befolgen. */
        <section className="mt-6 rounded-lg border border-slate-400 bg-white p-4">
          <h2 className="font-semibold">Dieser Wert ist vorläufig</h2>
          <p className="mt-2 text-slate-700">
            Er stützt sich auf {detail.pages}{' '}
            {detail.pages === 1 ? 'geprüfte Seite' : 'geprüfte Seiten'} und beschreibt damit diese
            Seite, nicht die Website. Unser Wert ist als gewichtetes Mittel über Startseite,
            rechtlich wichtige Seiten und den Rest definiert — dafür braucht es mehr.
          </p>
          <p className="mt-2 text-sm text-slate-600">
            Meist liegt es an der <code>robots.txt</code>: Verlangt sie drei Minuten Pause zwischen
            zwei Anfragen, halten wir uns daran, und in der verfügbaren Zeit bleiben wenige Seiten.
            Wir umgehen das nicht.
          </p>
        </section>
      )}

      {detail.obscured && (
        /* Ein sauberes Banner vor einer ungeprüften Seite ergibt eine gute Note. Das
           steht sonst zwei Bildschirme tiefer, und wer nur die Note liest, nimmt das
           Falsche mit. */
        <section className="mt-6 rounded-lg border border-slate-400 bg-white p-4">
          <h2 className="font-semibold">Gemessen wurde überwiegend das Einwilligungsbanner</h2>
          <p className="mt-2 text-slate-700">
            Auf den meisten geprüften Seiten ließ sich die Abfrage nach Cookies nicht schließen. Was
            dahinter liegt, haben wir nicht gesehen — die gefundenen Barrieren sind die des Banners.
          </p>
          <p className="mt-2 text-sm text-slate-600">
            Deshalb sagt dieser Wert wenig über die Website. Er kann in beide Richtungen falsch
            sein: Ein sauber gebautes Banner ergibt eine gute Note, obwohl niemand weiß, wie es
            dahinter aussieht.
          </p>
        </section>
      )}

      {detail.failure && (
        /* Eine abgewiesene Behörde sieht sonst aus wie eine, an die noch niemand
           herangekommen ist — und der Verdacht fiele auf sie statt auf die Sperre. */
        <section className="mt-6 rounded-lg border border-slate-400 bg-white p-4">
          <h2 className="font-semibold">Diese Website konnte nicht geprüft werden</h2>
          <p className="mt-2 text-slate-700">
            Beim letzten Versuch am {formatDate(detail.failure.at)} kamen wir nicht an die Inhalte
            heran: <span className="break-words">{detail.failure.reason}</span>
          </p>
          <p className="mt-2 text-sm text-slate-600">
            Das ist keine Aussage über die Barrierefreiheit dieser Seite. Ein Schutz gegen
            automatische Zugriffe hält auch uns fern — und eine Note aus einer Sperrseite wäre ein
            Urteil über unseren Prüfer, nicht über die Behörde.
          </p>
        </section>
      )}

      {scan.data?.statement && <StatementCheck statement={scan.data.statement} />}

      {scan.data?.third_parties && <ThirdParties contacts={scan.data.third_parties} />}

      {detail.mail && <MailRecord mail={detail.mail} />}

      {scan.data?.explanation && scan.data.explanation.improvements.length > 0 && (
        <section className="mt-8">
          <h2 className="text-xl font-semibold">Was am meisten bringt</h2>
          <p className="mt-2 text-slate-700">
            Die Befunde in der Reihenfolge, in der sie den Wert dieser Behörde am stärksten heben —
            gerechnet über alle geprüften Seiten und ihr jeweiliges Gewicht.
          </p>
          <ol className="mt-3 space-y-3">
            {scan.data.explanation.improvements.slice(0, 8).map((improvement) => (
              <li
                key={improvement.rule_id}
                className="rounded-lg border border-slate-200 bg-white p-4"
              >
                <div className="flex flex-wrap items-baseline justify-between gap-2">
                  <h3 className="font-semibold break-words hyphens-auto">
                    {improvement.help_url ? (
                      <a href={improvement.help_url} className="underline">
                        {regelName(improvement)}
                      </a>
                    ) : (
                      regelName(improvement)
                    )}
                  </h3>
                  <span className="font-semibold whitespace-nowrap">
                    +
                    {improvement.points_if_fixed.toLocaleString('de-DE', {
                      maximumFractionDigits: 1,
                    })}{' '}
                    Punkte
                  </span>
                </div>
                {regelWirkung(improvement.rule_id) && (
                  <p className="mt-1 text-slate-700">{regelWirkung(improvement.rule_id)}</p>
                )}
                <p className="mt-1 text-sm text-slate-700">
                  {impactLabel[improvement.impact]} · auf {improvement.pages}{' '}
                  {improvement.pages === 1 ? 'Seite' : 'Seiten'}, {improvement.nodes}{' '}
                  {improvement.nodes === 1 ? 'Element' : 'Elemente'} betroffen
                </p>
              </li>
            ))}
          </ol>
        </section>
      )}

      {detail.lighthouse_score !== undefined && (
        <section className="mt-6 rounded-lg border border-slate-200 bg-white p-6">
          <h2 className="text-xl font-semibold">Zum Vergleich: Lighthouse</h2>
          <p className="mt-2 text-2xl font-semibold">
            {formatScore(detail.lighthouse_score)}
            <span className="sr-only"> von 100 Punkten</span>
          </p>
          {/* Zwei Zahlen, die absichtlich verschieden rechnen: Lighthouse lässt ein
              Audit ganz durchfallen, sobald ein Element es verletzt, und gewichtet
              danach. Wir gewichten nach Schwere und dämpfen die Menge. Wo beide weit
              auseinanderliegen, lohnt der Blick in die Befunde. */}
          <p className="mt-2 text-slate-700">
            Googles Wert für die Startseite. Er bewertet jede Regel ganz oder gar nicht: ein
            fehlender Alternativtext unter hundert Bildern lässt die ganze Regel durchfallen. Unser
            Wert gewichtet nach Schwere und dämpft die Menge, betrachtet dafür mehrere Seiten. Zwei
            Blickwinkel auf dieselbe Website.
          </p>
        </section>
      )}

      {detail.score !== null && (
        <>
          <h2 className="mt-10 text-xl font-semibold">Nach WCAG-Prinzip</h2>
          <ul className="mt-3 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {(Object.keys(detail.subscores) as (keyof Subscores)[]).map((key) => (
              <li key={key} className="rounded-lg border border-slate-200 bg-white p-4">
                <h3 className="text-sm font-medium text-slate-700">{principleLabel[key]}</h3>
                <p className="mt-1 text-2xl font-semibold">{formatScore(detail.subscores[key])}</p>
                <p className="mt-2 text-sm text-slate-700">{principleExplanation[key]}</p>
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
               Befund über das Banner. Wer den Wert liest, muss das wissen. */
            <p className="mt-2 rounded-lg border border-grade-d bg-white p-4">
              Auf {scan.data.pages_blocked} {scan.data.pages_blocked === 1 ? 'Seite' : 'Seiten'}{' '}
              ließ sich die Einwilligungsabfrage nicht schließen. Dort beschreibt das Ergebnis das
              Banner und nicht die Seite dahinter.
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
              verdeckt genau die Spalte mit dem Wert. */}
          <ul className="mt-3 space-y-3 sm:hidden">
            {(scan.data.pages ?? []).map((page) => (
              <li key={page.url} className="rounded-lg border border-slate-200 bg-white p-4">
                <a href={page.url} className="block break-words py-1 underline">
                  {page.title || page.url}
                </a>
                <p className="mt-1 text-sm break-all text-slate-600">{page.url}</p>
                <p className="mt-2 text-sm">
                  Wert {formatScore(page.score)} · {page.violations}{' '}
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
              <caption className="sr-only">Die geprüften Seiten mit ihrem Wert</caption>
              <thead>
                <tr className="border-b border-slate-300">
                  <th
                    scope="col"
                    className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
                  >
                    Seite
                  </th>
                  <th
                    scope="col"
                    className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
                  >
                    Wert
                  </th>
                  <th
                    scope="col"
                    className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
                  >
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

          <h3 className="mt-8 text-lg font-semibold">Begründung je Seite</h3>
          <ul className="mt-3 space-y-2">
            {(scan.data.explanation?.pages ?? []).map((explained) => (
              <li key={explained.url} className="rounded-lg border border-slate-200 bg-white">
                <details>
                  <summary className="min-h-11 cursor-pointer px-4 py-3">
                    <span className="break-all">{explained.title || explained.url}</span>
                    <span className="ml-2 font-semibold whitespace-nowrap">
                      {formatScore(explained.score)}
                    </span>
                  </summary>
                  <div className="border-t border-slate-200 px-4 py-3">
                    <ScoreReasons page={explained} />
                  </div>
                </details>
              </li>
            ))}
          </ul>
        </>
      )}
    </>
  )
}
