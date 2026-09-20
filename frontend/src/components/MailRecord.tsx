import type { MailRecord as Record } from '../api/types'
import { dmarcLabel, formatDate, mailProviderLabel } from '../lib'

/**
 * Was die Domain einer Behörde über ihre E-Mail veröffentlicht. Öffentliches DNS,
 * nichts wird abgeklopft — und nichts davon zählt in den Score.
 *
 * Neben jeder Einordnung steht der Roheintrag. Ein MX-Eintrag sagt, welcher Host die
 * Post annimmt, nicht wer sie liest; ein SPF-Eintrag sagt, wer im Namen der Domain
 * senden darf, und gerade nicht, wo die Postfächer liegen.
 */
export function MailRecord({ mail }: { mail: Record }) {
  return (
    <section className="mt-10">
      <h2 className="text-xl font-semibold">Wohin die E-Mail geht</h2>

      {mail.error ? (
        <p className="mt-2 rounded-lg border border-slate-400 bg-white p-4">
          Die DNS-Abfrage für <code>{mail.domain}</code> blieb ohne Antwort. Über die E-Mail dieser
          Behörde sagen wir deshalb nichts.
        </p>
      ) : (
        <>
          <p className="mt-2 text-slate-700">
            Die Post an <code>{mail.domain}</code> nimmt entgegen:{' '}
            <strong>{mailProviderLabel[mail.provider]}</strong>
            {mail.checked_at && <> · abgefragt am {formatDate(mail.checked_at)}</>}
          </p>

          {mail.filter && (
            <p className="mt-2 rounded-lg border border-slate-400 bg-white p-4">
              Dieser Host ist ein vorgeschalteter Spamfilter. Er sagt, wo die Post geprüft wird — wo
              die Postfächer liegen, steht nicht im DNS.
            </p>
          )}

          {mail.mx && mail.mx.length > 0 && (
            <div className="mt-4 rounded-lg border border-slate-200 bg-white p-4">
              <h3 className="font-semibold">MX-Einträge</h3>
              <ul className="mt-2 space-y-1">
                {mail.mx.map((exchanger) => (
                  <li key={exchanger.host} className="text-sm">
                    <code className="break-all">{exchanger.host}</code>{' '}
                    <span className="text-slate-600">
                      (Priorität {exchanger.preference}
                      {exchanger.provider !== 'unknown' &&
                        ` · ${mailProviderLabel[exchanger.provider]}`}
                      )
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {mail.spf && (
            <div className="mt-4 rounded-lg border border-slate-200 bg-white p-4">
              <h3 className="font-semibold">SPF-Eintrag</h3>
              <p className="mt-1 text-sm text-slate-600">
                Er sagt, wer im Namen dieser Domain senden darf — nicht, wo die Postfächer liegen.
                Ein Newsletter-Dienst steht hier, ohne je ein Postfach zu sehen.
              </p>
              <p className="mt-2 text-sm break-all">
                <code>{mail.spf}</code>
              </p>
            </div>
          )}

          {mail.dmarc_policy && (
            <div className="mt-4 rounded-lg border border-slate-200 bg-white p-4">
              <h3 className="font-semibold">DMARC</h3>
              <p className="mt-1 text-sm text-slate-700">
                Gefälschte Absender in dieser Domain sollen Empfänger{' '}
                {dmarcLabel[mail.dmarc_policy] ?? mail.dmarc_policy} ({mail.dmarc_policy}).
              </p>
            </div>
          )}
        </>
      )}

      <p className="mt-4 text-sm text-slate-600">
        Gelesen wird ausschließlich öffentliches DNS. Ein MX-Eintrag ist kein Urteil: Dahinter kann
        ein deutscher Dienstleister stehen, der selbst bei einem US-Anbieter liegt, und er kann ein
        Spamfilter sein, während die Postfächer woanders stehen.
      </p>
    </section>
  )
}
