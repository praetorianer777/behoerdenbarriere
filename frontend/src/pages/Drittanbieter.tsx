import { useQuery } from '@tanstack/react-query'

import { api } from '../api/client'
import { Loading, LoadError } from '../components/Loading'
import { groupLabel, phaseExplanation, phaseLabel } from '../lib'
import type { ContactGroup, ContactPhase, ThirdPartyReach } from '../api/types'

const phases: ContactPhase[] = ['before_consent', 'after_declined', 'after_accepted']

interface GroupRow {
  group: ContactGroup
  agencies: number
  hosts: string[]
}

/**
 * Fasst die Hosts einer Phase zu den Diensten zusammen, unter deren Namen man sie
 * kennt. Für die Zahl der Behörden wird der größte Host des Dienstes genommen und
 * nicht summiert: Eine Behörde, die drei Google-Hosts anspricht, ist eine Behörde.
 * Die genaue Vereinigungsmenge kennt nur die Datenbank — diese Zahl ist die
 * vorsichtige Untergrenze.
 */
function byGroup(items: ThirdPartyReach[]): GroupRow[] {
  const rows = new Map<ContactGroup, GroupRow>()
  for (const item of items) {
    const row = rows.get(item.group) ?? { group: item.group, agencies: 0, hosts: [] }
    row.agencies = Math.max(row.agencies, item.agencies)
    row.hosts.push(item.host)
    rows.set(item.group, row)
  }
  return [...rows.values()].sort((a, b) => b.agencies - a.agencies)
}

export function Drittanbieter() {
  const data = useQuery({
    queryKey: ['thirdparties'],
    queryFn: ({ signal }) => api.thirdParties(signal),
  })

  if (data.isPending) return <Loading what="Die Auswertung der Drittanbieter" />
  if (data.isError) return <LoadError what="Die Auswertung der Drittanbieter" error={data.error} />

  const { items, scanned } = data.data
  const share = (agencies: number) => (scanned > 0 ? Math.round((agencies / scanned) * 100) : 0)

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-bold sm:text-3xl">Drittanbieter auf Behördenseiten</h1>
      <p className="mt-2 text-slate-700">
        Beim Prüfen einer Seite sehen wir, welche fremden Hosts der Browser kontaktiert — und wann.
        Jeder dieser Aufrufe überträgt die IP-Adresse der Besuchenden. Grundlage sind die jeweils
        neuesten Prüfungen von {scanned.toLocaleString('de-DE')} Behörden.
      </p>
      <p className="mt-2 text-slate-700">
        Das hat mit Barrierefreiheit nichts zu tun und zählt deshalb nicht in den Score.
      </p>

      {items.length === 0 && (
        <p className="mt-6 rounded-lg border border-slate-200 bg-white p-4">
          Es liegen noch keine Beobachtungen vor.
        </p>
      )}

      {phases.map((phase) => {
        const rows = byGroup(items.filter((item) => item.phase === phase))
        if (rows.length === 0) return null

        return (
          <section key={phase} className="mt-8">
            <h2 className="text-xl font-semibold">{phaseLabel[phase]}</h2>
            <p className="mt-1 text-sm text-slate-600">{phaseExplanation[phase]}</p>

            <table className="mt-3 w-full border-collapse text-left">
              <caption className="sr-only">
                Dienste, die {phaseLabel[phase]} kontaktiert wurden, mit der Zahl der betroffenen
                Behörden
              </caption>
              <thead>
                <tr className="border-b border-slate-300">
                  <th scope="col" className="py-2 pr-4">
                    Dienst
                  </th>
                  <th scope="col" className="py-2 pr-4">
                    Behörden
                  </th>
                  <th scope="col" className="py-2">
                    Anteil
                  </th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => (
                  <tr key={row.group} className="border-b border-slate-200 align-top">
                    <th scope="row" className="py-2 pr-4 font-medium">
                      {groupLabel[row.group]}
                      {/* Der Rohwert gehört dazu: Wer die Einordnung prüfen will,
                          muss den Hostnamen sehen. */}
                      <span className="mt-1 block text-sm font-normal break-all text-slate-600">
                        {row.hosts.slice(0, 4).join(', ')}
                        {row.hosts.length > 4 && ` und ${row.hosts.length - 4} weitere`}
                      </span>
                    </th>
                    <td className="py-2 pr-4 whitespace-nowrap">{row.agencies}</td>
                    <td className="py-2 whitespace-nowrap">{share(row.agencies)} %</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>
        )
      })}

      <section className="mt-10 rounded-lg border border-slate-200 bg-white p-4 text-sm text-slate-700">
        <h2 className="font-semibold">Was diese Zahlen nicht sagen</h2>
        <ul className="mt-2 list-disc space-y-1 pl-5">
          <li>
            Ein Hostname sagt, wohin der Browser eine Anfrage schickt — nicht, wer die Daten am Ende
            verarbeitet. Ein CDN kann im Auftrag der Behörde arbeiten.
          </li>
          <li>
            Wir rufen jede Seite einmal auf. Einbettungen, die erst nach einem Klick laden, sehen
            wir nicht.
          </li>
          <li>
            „Vor der Einwilligung“ umfasst auch Seiten, die gar nicht erst fragen. Dass gefragt
            wurde, ist dabei nicht der Maßstab — dass etwas geladen wurde, bevor jemand
            widersprechen konnte, ist es.
          </li>
          <li>
            Empfänger, die selbst öffentliche Stellen sind, stehen mit in den Listen der einzelnen
            Behörden, sind dort aber als solche gekennzeichnet.
          </li>
        </ul>
      </section>
    </div>
  )
}
