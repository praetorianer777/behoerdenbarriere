import { useQuery } from '@tanstack/react-query'

import { api } from '../api/client'
import { Loading, LoadError } from '../components/Loading'
import { formatDate, levelLabel, mailProviderLabel } from '../lib'
import type { MailCount } from '../api/types'

/**
 * Die Kacheln stehen an ungefähr der Stelle, an der das Land auf der Karte liegt. Sie
 * bilden keine Flächen ab — ein Kartogramm ist eine Leseordnung, keine Landkarte, und
 * eine verzerrte Karte würde mehr behaupten als die Daten hergeben.
 */
const tiles: { state: string; short: string; col: number; row: number }[] = [
  { state: 'Schleswig-Holstein', short: 'SH', col: 3, row: 1 },
  { state: 'Mecklenburg-Vorpommern', short: 'MV', col: 5, row: 1 },
  { state: 'Bremen', short: 'HB', col: 2, row: 2 },
  { state: 'Hamburg', short: 'HH', col: 3, row: 2 },
  { state: 'Niedersachsen', short: 'NI', col: 4, row: 2 },
  { state: 'Brandenburg', short: 'BB', col: 5, row: 2 },
  { state: 'Berlin', short: 'BE', col: 6, row: 2 },
  { state: 'Nordrhein-Westfalen', short: 'NW', col: 3, row: 3 },
  { state: 'Sachsen-Anhalt', short: 'ST', col: 4, row: 3 },
  { state: 'Sachsen', short: 'SN', col: 5, row: 3 },
  { state: 'Rheinland-Pfalz', short: 'RP', col: 3, row: 4 },
  { state: 'Hessen', short: 'HE', col: 4, row: 4 },
  { state: 'Thüringen', short: 'TH', col: 5, row: 4 },
  { state: 'Saarland', short: 'SL', col: 2, row: 5 },
  { state: 'Baden-Württemberg', short: 'BW', col: 3, row: 5 },
  { state: 'Bayern', short: 'BY', col: 4, row: 5 },
]

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
      <h1 className="text-2xl font-bold sm:text-3xl">Wohin die Post der Behörden geht</h1>
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

            {/* Die Kacheln wiederholen nur die Tabelle darunter. Für Screenreader
                zählt die Tabelle, deshalb ist das Raster hier ausgeblendet. */}
            <div
              aria-hidden="true"
              className="mt-4 grid grid-cols-6 gap-1 sm:gap-2"
              style={{ gridTemplateRows: 'repeat(5, minmax(0, 1fr))' }}
            >
              {tiles.map((tile) => {
                const share = byState.get(tile.state)
                return (
                  <div
                    key={tile.short}
                    style={{ gridColumn: tile.col, gridRow: tile.row }}
                    className={`rounded-md border border-slate-300 p-2 text-center ${tileClass(share)}`}
                  >
                    <span className="block text-xs font-medium">{tile.short}</span>
                    <span className="block text-sm font-semibold">
                      {share && share.total > 0 ? `${share.percent} %` : '–'}
                    </span>
                  </div>
                )
              })}
            </div>

            <table className="mt-6 w-full border-collapse text-left">
              <caption className="sr-only">
                Anteil der Behörden je Bundesland, deren E-Mail bei einem Anbieter mit Sitz in den
                USA eingeht
              </caption>
              <thead>
                <tr className="border-b border-slate-300">
                  <th scope="col" className="py-2 pr-4">
                    Bundesland
                  </th>
                  <th scope="col" className="py-2 pr-4">
                    Behörden
                  </th>
                  <th scope="col" className="py-2">
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
                  <th scope="col" className="py-2 pr-4">
                    Anbieter
                  </th>
                  <th scope="col" className="py-2 pr-4">
                    Behörden
                  </th>
                  <th scope="col" className="py-2">
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
            Mit Barrierefreiheit hat all das nichts zu tun. In den Score fließt nichts davon ein.
          </li>
        </ul>
      </section>
    </div>
  )
}
