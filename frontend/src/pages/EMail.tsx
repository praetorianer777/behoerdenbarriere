import { useQuery } from '@tanstack/react-query'

import { api } from '../api/client'
import { Kartogramm } from '../components/Kartogramm'
import { laender as tiles } from '../laender'
import { Loading, LoadError } from '../components/Loading'
import { formatDate, levelLabel, mailProviderLabel } from '../lib'
import type { MailCount } from '../api/types'

interface Share {
  total: number
  us: number
  percent: number
}

function shares(counts: MailCount[]): Map<string, Share> {
  const out = new Map<string, Share>()
  for (const count of counts) {
    const key = count.name ?? ''
    const share = out.get(key) ?? { total: 0, us: 0, percent: 0 }
    share.total += count.agencies
    if (count.us_based) share.us += count.agencies
    share.percent = share.total > 0 ? Math.round((share.us / share.total) * 100) : 0
    out.set(key, share)
  }
  return out
}

// Vier Stufen, und in jeder Kachel steht die Zahl. Die Farbe ordnet, sie trägt die
// Aussage nicht: Wer sie nicht unterscheiden kann, liest den Prozentwert.
function tileClass(share: Share | undefined): string {
  if (!share || share.total === 0) return 'bg-white text-slate-500 border-dashed'
  if (share.percent === 0) return 'bg-white text-slate-900'
  if (share.percent < 25) return 'bg-slate-200 text-slate-900'
  if (share.percent < 50) return 'bg-slate-400 text-slate-900'
  return 'bg-slate-700 text-white'
}

export function EMail() {
  const data = useQuery({ queryKey: ['mail'], queryFn: ({ signal }) => api.mail(signal) })

  if (data.isPending) return <Loading what="Die Auswertung der DNS-Einträge" />
  if (data.isError) return <LoadError what="Die Auswertung der DNS-Einträge" error={data.error} />

  const summary = data.data
  const byState = shares(summary.by_state)
  const usTotal = summary.by_provider
    .filter((count) => count.us_based)
    .reduce((sum, count) => sum + count.agencies, 0)
  const usPercent = summary.total > 0 ? Math.round((usTotal / summary.total) * 100) : 0

  return (
    <div className="max-w-3xl">
      <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">
        Wohin die Post der Behörden geht
      </h1>
      <p className="mt-2 text-slate-700">
        Jede Domain veröffentlicht im DNS, welcher Host ihre E-Mail entgegennimmt. Daraus lässt sich
        ablesen, über wessen Server die Korrespondenz einer Behörde läuft. Wir lesen nur, was
        veröffentlicht ist — kein Portscan, keine Anfrage an einen Mailserver.
      </p>

      {summary.total === 0 ? (
        <p className="mt-6 rounded-lg border border-slate-200 bg-white p-4">
          Es liegen noch keine Abfragen vor.
        </p>
      ) : (
        <>
          <p className="mt-4 rounded-lg border border-slate-200 bg-white p-4">
            Von {summary.total.toLocaleString('de-DE')} abgefragten Behörden nehmen{' '}
            <strong>{usTotal.toLocaleString('de-DE')}</strong> ({usPercent} %) ihre E-Mail über
            einen Anbieter mit Sitz in den USA entgegen.
            {summary.checked_at && <> Letzte Abfrage: {formatDate(summary.checked_at)}.</>}
          </p>

          <section className="mt-8">
            <h2 className="text-xl font-semibold">Nach Bundesland</h2>
            <p className="mt-1 text-sm text-slate-600">
              Anteil der Behörden je Land, deren Post bei einem US-Anbieter eingeht. Die Karte ist
              schematisch: Jede Kachel ist ein Land, kein Land ist so groß wie seine Kachel.
            </p>

            <Kartogramm
              werte={byState}
              text={(share) => (share && share.total > 0 ? `${share.percent} %` : '–')}
              klasse={tileClass}
            />

            <table className="mt-6 w-full border-collapse text-left">
              <caption className="sr-only">
                Anteil der Behörden je Bundesland, deren E-Mail bei einem Anbieter mit Sitz in den
                USA eingeht
              </caption>
              <thead>
                <tr className="border-b border-slate-300">
                  <th
                    scope="col"
                    className="py-2 pr-4 text-xs font-semibold tracking-wide text-slate-600 uppercase"
                  >
                    Bundesland
                  </th>
                  <th
                    scope="col"
                    className="py-2 pr-4 text-xs font-semibold tracking-wide text-slate-600 uppercase"
                  >
                    Behörden
                  </th>
                  <th
                    scope="col"
                    className="py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
                  >
                    davon US-Anbieter
                  </th>
                </tr>
              </thead>
              <tbody>
                {tiles
                  .map((tile) => ({ tile, share: byState.get(tile.state) }))
                  .filter((row) => row.share && row.share.total > 0)
                  .sort((a, b) => b.share!.percent - a.share!.percent)
                  .map(({ tile, share }) => (
                    <tr key={tile.short} className="border-b border-slate-200">
                      <th scope="row" className="py-2 pr-4 font-medium">
                        {tile.state}
                      </th>
                      <td className="py-2 pr-4">{share!.total}</td>
                      <td className="py-2">
                        {share!.us} ({share!.percent} %)
                      </td>
                    </tr>
                  ))}
              </tbody>
            </table>
          </section>

          <section className="mt-8">
            <h2 className="text-xl font-semibold">Nach Anbieter</h2>
            <table className="mt-3 w-full border-collapse text-left">
              <caption className="sr-only">
                Zahl der Behörden je Anbieter, der ihre E-Mail entgegennimmt
              </caption>
              <thead>
                <tr className="border-b border-slate-300">
                  <th
                    scope="col"
                    className="py-2 pr-4 text-xs font-semibold tracking-wide text-slate-600 uppercase"
                  >
                    Anbieter
                  </th>
                  <th
                    scope="col"
                    className="py-2 pr-4 text-xs font-semibold tracking-wide text-slate-600 uppercase"
                  >
                    Behörden
                  </th>
                  <th
                    scope="col"
                    className="py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
                  >
                    Sitz
                  </th>
                </tr>
              </thead>
              <tbody>
                {summary.by_provider.map((count) => (
                  <tr key={count.provider} className="border-b border-slate-200">
                    <th scope="row" className="py-2 pr-4 font-medium">
                      {mailProviderLabel[count.provider]}
                    </th>
                    <td className="py-2 pr-4">{count.agencies}</td>
                    <td className="py-2">
                      {count.us_based ? 'USA' : '—'}
                      {count.filter && (
                        <span className="block text-sm text-slate-600">
                          vorgeschalteter Spamfilter
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>

          <section className="mt-8">
            <h2 className="text-xl font-semibold">Nach Ebene</h2>
            <ul className="mt-3 space-y-1">
              {summary.by_level.map((count) => (
                <li key={`${count.name}-${count.provider}`} className="text-slate-700">
                  {levelLabel[count.name ?? ''] ?? count.name}: {count.agencies} ×{' '}
                  {mailProviderLabel[count.provider]}
                </li>
              ))}
            </ul>
          </section>
        </>
      )}

      <section className="mt-10 rounded-lg border border-slate-200 bg-white p-4 text-sm text-slate-700">
        <h2 className="font-semibold">Was ein MX-Eintrag nicht sagt</h2>
        <ul className="mt-2 list-disc space-y-1 pl-5">
          <li>
            Er sagt, welcher Host die Post annimmt — nicht, wer sie am Ende liest. Dahinter kann ein
            deutscher Dienstleister stehen, der selbst bei einem US-Anbieter liegt.
          </li>
          <li>Er kann auf einen Spamfilter zeigen, während die Postfächer ganz woanders stehen.</li>
          <li>
            Ein SPF-Eintrag ist kein Mail-Hosting. Er erlaubt einem Dienst, im Namen der Domain zu
            senden — einem Newsletter-Werkzeug etwa. Wir führen ihn deshalb getrennt und nur auf der
            Seite der jeweiligen Behörde.
          </li>
          <li>
            „Sitz USA“ ist eine Aussage über das Unternehmen, nicht über den Standort eines Servers
            — und für sich genommen keine rechtliche Bewertung.
          </li>
          <li>
            Mit Barrierefreiheit hat all das nichts zu tun. In den Wert fließt nichts davon ein.
          </li>
        </ul>
      </section>
    </div>
  )
}
