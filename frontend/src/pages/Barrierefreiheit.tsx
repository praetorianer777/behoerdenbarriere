import { betreiber } from '../betreiber'
import { Platzhalterhinweis } from '../components/Platzhalterhinweis'
import { formatDate } from '../lib'

/**
 * Unsere eigene Erklärung nach § 12b BGG. Sie enthält dieselben Pflichtangaben, die
 * wir bei 426 Behörden prüfen — und wird in der CI mit demselben Prüfprogramm geprüft.
 * Wer Transparenz verlangt, kommt daran nicht vorbei.
 */
export function Barrierefreiheit() {
  return (
    <div className="max-w-3xl">
      <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">
        Erklärung zur Barrierefreiheit
      </h1>
      <Platzhalterhinweis />

      <p className="mt-4">
        Diese Erklärung gilt für die Website behoerdenbarriere.de. Wir sind keine öffentliche Stelle
        und deshalb nicht verpflichtet, eine solche Erklärung abzugeben. Wir tun es trotzdem: Wer
        426 Behörden daraufhin prüft, sollte selbst nachweisen, woran er sich messen lässt.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Stand der Vereinbarkeit</h2>
      <p className="mt-2">
        Diese Website ist mit der Barrierefreie-Informationstechnik-Verordnung (BITV 2.0) und den
        Anforderungen der WCAG 2.1 auf Stufe AA <strong>teilweise vereinbar</strong>. Die maschinell
        prüfbaren Anforderungen werden erfüllt; eine vollständige Konformität behaupten wir nicht,
        weil eine manuelle Prüfung durch eine anerkannte Prüfstelle bisher nicht stattgefunden hat.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Nicht barrierefreie Inhalte</h2>
      <p className="mt-2">Folgende Bereiche sind nicht oder nur eingeschränkt zugänglich:</p>
      <ul className="mt-2 list-disc space-y-2 pl-6">
        <li>
          Die Verlaufsdiagramme sind Grafiken ohne eigene Auszeichnung. Alle Werte stehen darunter
          als Tabelle, aber die Kurve selbst ist für Screenreader nicht lesbar.
        </li>
        <li>
          Die HTML-Ausschnitte in den Befunden sind unformatierter Quelltext. Vorgelesen ergeben sie
          wenig Sinn; sie richten sich an Personen, die die betroffene Seite reparieren.
        </li>
        <li>
          Breite Tabellen lassen sich auf schmalen Bildschirmen waagerecht scrollen. Das ist nach
          WCAG zulässig, aber unbequem.
        </li>
        <li>
          Geprüft wurde bisher automatisiert und mit Tastatur, nicht mit Screenreadern durch
          Nutzerinnen und Nutzer mit Behinderung. Was diesen Prüfungen entgeht, wissen wir nicht.
        </li>
      </ul>

      <h2 className="mt-8 text-xl font-semibold">Erstellung dieser Erklärung</h2>
      <p className="mt-2">
        Diese Erklärung wurde am {formatDate(betreiber.barrierefreiheitGeprueftAm)} erstellt und
        zuletzt überprüft. Grundlage war eine Selbstbewertung: Jede Seite wird bei jeder Änderung
        automatisiert mit axe-core gegen WCAG 2.1 AA geprüft — mit demselben Prüfprogramm, das wir
        auf Behördenseiten anwenden — und zusätzlich in drei Bildschirmbreiten sowie mit der
        Tastatur getestet.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Barrieren melden</h2>
      <p className="mt-2">
        Sind Ihnen Mängel beim barrierefreien Zugang aufgefallen? Schreiben Sie uns an{' '}
        <a href={`mailto:${betreiber.email}`} className="break-all underline">
          {betreiber.email}
        </a>
        . Wir antworten, so schnell wir können, und sagen Ihnen, was wir ändern — oder warum nicht.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Schlichtungsverfahren</h2>
      <p className="mt-2">
        Für öffentliche Stellen gilt: Wenn auf eine Meldung keine zufriedenstellende Antwort folgt,
        kann die Schlichtungsstelle nach § 16 BGG angerufen werden. Für uns als privates Angebot ist
        dieses Verfahren nicht eröffnet — wir nennen es hier, weil es zu den Pflichtangaben gehört,
        die wir bei Behörden prüfen:
      </p>
      <address className="mt-2 not-italic">
        Schlichtungsstelle nach dem Behindertengleichstellungsgesetz
        <br />
        bei dem Beauftragten der Bundesregierung für die Belange von Menschen mit Behinderungen
        <br />
        Mauerstraße 53, 10117 Berlin
        <br />
        <a href="mailto:info@schlichtungsstelle-bgg.de" className="underline">
          info@schlichtungsstelle-bgg.de
        </a>
      </address>
    </div>
  )
}
