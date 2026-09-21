import { useQuery } from '@tanstack/react-query'

import { api } from '../api/client'
import { Loading, LoadError } from '../components/Loading'
import { formatDate } from '../lib'
import type { UsageKey } from '../api/types'

const windowDays = 30

const pageLabel: Record<string, string> = {
  '/': 'Ranking (Startseite)',
  '/dashboard': 'Überblick',
  '/methodik': 'Methodik',
  '/statistik': 'Statistik',
  '/drittanbieter': 'Drittanbieter',
  '/e-mail': 'E-Mail-Auswertung',
  '/behoerde/:slug': 'Behördenseiten',
}

function formatCount(value: number): string {
  return value.toLocaleString('de-DE')
}

export function Statistics() {
  const usage = useQuery({
    queryKey: ['usage', windowDays],
    queryFn: ({ signal }) => api.usage(windowDays, signal),
  })

  if (usage.isPending) return <Loading what="Die Nutzungsstatistik" />
  if (usage.isError) return <LoadError what="Die Nutzungsstatistik" error={usage.error} />

  const data = usage.data
  const maxVisitors = Math.max(1, ...data.days.map((day) => day.visitors))

  return (
    <div className="max-w-3xl">
      <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">Nutzung dieser Seite</h1>
      <p className="mt-2 text-slate-700">
        Wir verlangen von Behörden Offenheit, also legen wir unsere eigenen Zahlen offen. Zeitraum:{' '}
        {formatDate(data.since)} bis {formatDate(data.until)}.
      </p>

      <ul className="mt-6 grid gap-4 sm:grid-cols-2">
        <li className="rounded-lg border border-slate-200 bg-white p-4">
          <h2 className="text-sm font-medium text-slate-700">Besuche</h2>
          <p className="mt-1 text-3xl font-semibold">{formatCount(data.visitors)}</p>
          <p className="mt-1 text-sm text-slate-700">Summe der Tageswerte</p>
        </li>
        <li className="rounded-lg border border-slate-200 bg-white p-4">
          <h2 className="text-sm font-medium text-slate-700">Seitenaufrufe</h2>
          <p className="mt-1 text-3xl font-semibold">{formatCount(data.views)}</p>
          <p className="mt-1 text-sm text-slate-700">im gesamten Zeitraum</p>
        </li>
      </ul>

      <h2 className="mt-10 text-xl font-semibold">Je Tag</h2>
      {data.days.length === 0 ? (
        <p className="mt-2 text-slate-700">Für diesen Zeitraum liegen noch keine Zahlen vor.</p>
      ) : (
        <div
          className="mt-3 overflow-x-auto"
          tabIndex={0}
          role="region"
          aria-label="Tabelle, waagerecht scrollbar"
        >
          <table className="w-full border-collapse bg-white text-left">
            <caption className="sr-only">Besuche und Seitenaufrufe je Tag</caption>
            <thead>
              <tr className="border-b border-slate-300">
                <th
                  scope="col"
                  className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
                >
                  Tag
                </th>
                <th
                  scope="col"
                  className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
                >
                  Besuche
                </th>
                <th
                  scope="col"
                  className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
                >
                  Seitenaufrufe
                </th>
                <th
                  scope="col"
                  className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
                >
                  Verlauf
                </th>
              </tr>
            </thead>
            <tbody>
              {data.days.map((day) => (
                <tr key={day.day} className="border-b border-slate-200">
                  <th scope="row" className="px-3 py-2 font-normal">
                    {formatDate(day.day)}
                  </th>
                  <td className="px-3 py-2">{formatCount(day.visitors)}</td>
                  <td className="px-3 py-2">{formatCount(day.views)}</td>
                  <td className="px-3 py-2">
                    {/* Der Balken bebildert nur, was die Zahl daneben schon sagt. */}
                    <span
                      aria-hidden="true"
                      className="inline-block h-3 rounded bg-slate-700"
                      style={{
                        width: `${(day.visitors / maxVisitors) * 100}%`,
                        minWidth: day.visitors ? '4px' : 0,
                      }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <h2 className="mt-10 text-xl font-semibold">Seiten</h2>
      <CountTable
        caption="Seitenaufrufe je Seite"
        head="Seite"
        rows={data.pages}
        label={(key) => pageLabel[key] ?? key}
        empty="Noch keine Aufrufe gezählt."
      />

      <h2 className="mt-10 text-xl font-semibold">Meistgesehene Behörden</h2>
      <CountTable
        caption="Aufrufe je Behördenseite"
        head="Behörde"
        rows={data.agencies}
        empty="Noch keine Behördenseite aufgerufen."
      />

      <h2 className="mt-10 text-xl font-semibold">API-Aufrufe</h2>
      <CountTable
        caption="Aufrufe je API-Endpunkt"
        head="Endpunkt"
        rows={data.endpoints}
        empty="Noch keine Aufrufe gezählt."
        mono
      />

      <h2 className="mt-10 text-xl font-semibold">Wie gezählt wird</h2>
      <p className="mt-2">
        Gezählt wird in unserer eigenen API, nicht mit einem fremden Skript. Es gibt keine Cookies,
        keine Kennung im Browser und keine Daten, die das Haus verlassen. Gespeichert werden
        ausschließlich Tageszähler der Form <em>Tag, Art, Schlüssel, Anzahl</em> — also zum Beispiel
        „am 19.09. wurde die Seite <code>/dashboard</code> 42-mal aufgerufen“. Ein einzelner Aufruf
        ist darin nicht mehr enthalten.
      </p>
      <p className="mt-4">
        Ein Besuch wird über einen Hash aus IP-Adresse und Browserkennung erkannt. Der Schlüssel
        dieses Hashes wird täglich neu zufällig gezogen, liegt nur im Arbeitsspeicher und wird
        nirgends gespeichert. Damit lässt sich der Hash weder zurückrechnen noch über Tage hinweg
        zusammenführen: dieselbe Person an zwei Tagen ergibt zwei Besuche. Die Tageswerte sind
        deshalb eine Obergrenze und keine Personenzahl. Die Hashes selbst werden nach {windowDays}{' '}
        Tagen gelöscht.
      </p>
      <p className="mt-4">
        Nicht gespeichert werden: IP-Adressen, Browserkennungen, woher jemand kommt, und alles, was
        zu einer einzelnen Person gehört. Offensichtliche Bots zählen nicht mit, und die Endpunkte
        der Statistik selbst zählen sich nicht.
      </p>
    </div>
  )
}

function CountTable({
  caption,
  head,
  rows,
  label,
  empty,
  mono,
}: {
  caption: string
  head: string
  rows: UsageKey[]
  label?: (key: string) => string
  empty: string
  mono?: boolean
}) {
  if (rows.length === 0) return <p className="mt-2 text-slate-700">{empty}</p>
  return (
    <div
      className="mt-3 overflow-x-auto"
      tabIndex={0}
      role="region"
      aria-label="Tabelle, waagerecht scrollbar"
    >
      <table className="w-full border-collapse bg-white text-left">
        <caption className="sr-only">{caption}</caption>
        <thead>
          <tr className="border-b border-slate-300">
            <th
              scope="col"
              className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
            >
              {head}
            </th>
            <th
              scope="col"
              className="px-3 py-2 text-xs font-semibold tracking-wide text-slate-600 uppercase"
            >
              Aufrufe
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.key} className="border-b border-slate-200">
              <th
                scope="row"
                className={`px-3 py-2 font-normal ${mono ? 'font-mono text-sm' : ''}`}
              >
                {label ? label(row.key) : row.key}
              </th>
              <td className="px-3 py-2">{formatCount(row.count)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
