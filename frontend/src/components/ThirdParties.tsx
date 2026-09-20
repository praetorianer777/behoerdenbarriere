import type { ContactPhase, ThirdParty } from '../api/types'
import { groupLabel, phaseExplanation, phaseLabel } from '../lib'

const phases: ContactPhase[] = ['before_consent', 'after_declined', 'after_accepted']

/**
 * Welche fremden Hosts eine Seite kontaktiert, ist eine Datenschutzfrage und keine
 * Frage der Barrierefreiheit. Der Abschnitt steht deshalb neben dem Score und fließt
 * nicht in ihn ein.
 *
 * Gezeigt wird, was beobachtet wurde: welcher Host, in welcher Phase des Besuchs, auf
 * wie vielen Seiten. Ein Hostname beweist nicht, wohin Daten am Ende fließen — die
 * Einordnung überlassen wir den Lesenden.
 */
export function ThirdParties({ contacts }: { contacts: ThirdParty[] }) {
  if (contacts.length === 0) return null

  const byPhase = phases
    .map((phase) => ({ phase, items: contacts.filter((c) => c.phase === phase) }))
    .filter((group) => group.items.length > 0)

  const beforeConsent = contacts.filter((c) => c.phase === 'before_consent' && !c.public_body)

  return (
    <section className="mt-10">
      <h2 className="text-xl font-semibold">Eingebundene Drittanbieter</h2>

      {beforeConsent.length > 0 ? (
        <p className="mt-2 rounded-lg border border-grade-d bg-white p-4">
          Beim bloßen Aufruf wurden {beforeConsent.length}{' '}
          {beforeConsent.length === 1 ? 'fremder Host' : 'fremde Hosts'} kontaktiert, bevor eine
          Einwilligung möglich war. Dabei wird die IP-Adresse der Besuchenden übertragen.
        </p>
      ) : (
        <p className="mt-2 text-slate-700">
          Vor einer Einwilligung wurde kein fremder Host kontaktiert.
        </p>
      )}

      {byPhase.map((group) => (
        <div key={group.phase} className="mt-6">
          <h3 className="font-semibold">{phaseLabel[group.phase]}</h3>
          <p className="mt-1 text-sm text-slate-600">{phaseExplanation[group.phase]}</p>

          <ul className="mt-3 space-y-2">
            {group.items.map((contact) => (
              <li
                key={`${contact.host}-${contact.phase}`}
                className="rounded-lg border border-slate-200 bg-white p-4"
              >
                <div className="flex flex-wrap items-baseline justify-between gap-2">
                  {/* Der Rohwert zuerst: Die Einordnung ist unsere Lesart, der
                      Hostname ist die Beobachtung. */}
                  <code className="font-semibold break-all">{contact.host}</code>
                  <span className="text-sm whitespace-nowrap text-slate-700">
                    auf {contact.pages} {contact.pages === 1 ? 'Seite' : 'Seiten'}
                  </span>
                </div>
                <p className="mt-1 text-sm text-slate-700">
                  {groupLabel[contact.group]}
                  {contact.public_body && ' · öffentliche Stelle'}
                  {contact.self_hosted && ' · selbst betrieben'}
                </p>
              </li>
            ))}
          </ul>
        </div>
      ))}

      <p className="mt-4 text-sm text-slate-600">
        Beobachtet wird der aufgerufene Hostname, nicht der Empfänger der Daten: Ein CDN kann im
        Auftrag der Behörde arbeiten, und manche Einbettungen laden erst nach einem Klick. Zu einem
        Verstoß wird daraus erst, wenn jemand den Einzelfall prüft.
      </p>
    </section>
  )
}
