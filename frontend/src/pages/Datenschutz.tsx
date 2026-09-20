import { betreiber } from '../betreiber'
import { Platzhalterhinweis } from '../components/Platzhalterhinweis'

export function Datenschutz() {
  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-bold sm:text-3xl">Datenschutzerklärung</h1>
      <Platzhalterhinweis />

      <p className="mt-4">
        Diese Website kommt ohne Cookies aus, bindet nichts von fremden Servern ein und setzt keine
        Kennungen, die Sie über Tage hinweg wiedererkennbar machen. Was trotzdem an Daten anfällt,
        steht hier vollständig.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Verantwortlich</h2>
      <address className="mt-2 not-italic">
        {betreiber.name}
        <br />
        {betreiber.strasse}
        <br />
        {betreiber.ort}
        <br />
        <a href={`mailto:${betreiber.email}`} className="break-all underline">
          {betreiber.email}
        </a>
      </address>

      <h2 className="mt-8 text-xl font-semibold">Aufrufe dieser Website</h2>
      <p className="mt-2">
        Wir zählen, wie oft welche Seite aufgerufen wird, und schätzen die Zahl der Besucherinnen
        und Besucher. Das geschieht auf unserem eigenen Server, ohne Cookies und ohne fremde
        Dienste.
      </p>
      <p className="mt-4">
        Dafür wird aus Ihrer IP-Adresse, Ihrer Browserkennung und einem Zufallswert ein Hashwert
        gebildet. Der Zufallswert wechselt täglich und wird nirgends gespeichert; er liegt nur im
        Arbeitsspeicher. Dadurch lässt sich derselbe Hashwert am nächsten Tag nicht mehr herstellen,
        und zwei Tage lassen sich nicht miteinander verbinden.{' '}
        <strong>Ihre IP-Adresse wird zu keinem Zeitpunkt gespeichert.</strong> Gespeichert wird nur,
        wie oft etwas an einem Tag vorkam.
      </p>
      <p className="mt-4">
        Rechtsgrundlage ist Artikel 6 Absatz 1 Buchstabe f DSGVO. Unser berechtigtes Interesse ist
        zu wissen, ob und wie dieses Angebot genutzt wird. Die Zählwerte werden nach 30 Tagen
        gelöscht. Die Zahlen stehen offen auf der Seite{' '}
        <a href="/statistik" className="underline">
          Statistik
        </a>{' '}
        — wir verlangen Transparenz von Behörden und halten uns selbst daran.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Server-Protokolle</h2>
      <p className="mt-2">
        Der Webserver protokolliert technisch bedingt jeden Abruf mit Datum, Uhrzeit, abgerufener
        Adresse, Browserkennung und IP-Adresse. Diese Protokolle dienen dem Betrieb und der Abwehr
        von Angriffen, werden nicht mit anderen Daten zusammengeführt und spätestens nach sieben
        Tagen gelöscht. Rechtsgrundlage ist ebenfalls Artikel 6 Absatz 1 Buchstabe f DSGVO.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Hosting</h2>
      <p className="mt-2">
        Die Website wird betrieben bei: {betreiber.hosting}. Der Anbieter verarbeitet die oben
        genannten Protokolldaten in unserem Auftrag.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Die Schnittstelle</h2>
      <p className="mt-2">
        Die Daten stehen auch über eine offene Schnittstelle bereit. Abrufe darüber werden je
        Endpunkt gezählt, mit denselben Regeln wie oben. Wer einen Zugangsschlüssel für größere
        Mengen nutzt, wird über diesen Schlüssel unterschieden statt über die IP-Adresse.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Was wir bei den geprüften Behörden tun</h2>
      <p className="mt-2">
        Unser Prüfprogramm ruft öffentlich erreichbare Seiten von Behörden ab, wie es ein Browser
        täte. Dabei werden keine Formulare abgeschickt, keine Anmeldungen vorgenommen und keine
        personenbezogenen Daten erhoben. Gespeichert werden die geprüften Adressen, die gefundenen
        Verstöße samt einem kurzen Ausschnitt des betroffenen HTML und die daraus errechneten
        Zahlen.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Ihre Rechte</h2>
      <p className="mt-2">
        Sie haben das Recht auf Auskunft, Berichtigung, Löschung, Einschränkung der Verarbeitung und
        Widerspruch sowie das Recht, sich bei einer Datenschutz-Aufsichtsbehörde zu beschweren. Da
        wir keine Daten speichern, die einer Person zugeordnet werden können, werden wir Sie in
        unseren Zählwerten allerdings nicht wiederfinden — das ist der Zweck des Verfahrens.
      </p>
    </div>
  )
}
