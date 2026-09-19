export function Methodology() {
  return (
    <div className="max-w-3xl">
      <h1 className="text-3xl font-bold">Wie wir prüfen</h1>

      <h2 className="mt-8 text-xl font-semibold">Was geprüft wird</h2>
      <p className="mt-2">
        Jede Seite wird in einem echten Browser geladen und mit{' '}
        <a href="https://github.com/dequelabs/axe-core" className="underline">
          axe-core
        </a>{' '}
        gegen WCAG 2.1 auf den Stufen A und AA geprüft — den Maßstab, den BITV 2.0 und EN 301 549
        für öffentliche Stellen setzen. Geprüft wird die Startseite und, soweit das Seitenbudget
        reicht, weitere Seiten desselben Auftritts. Vorrang haben die Seiten, auf die es rechtlich
        ankommt: die Erklärung zur Barrierefreiheit, Kontakt, Formulare und Angebote in Leichter
        Sprache oder Gebärdensprache.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Wie gerechnet wird</h2>
      <p className="mt-2">Jeder gefundene Verstoß bekommt ein Gewicht nach seiner Schwere:</p>
      <table className="mt-4 w-full max-w-md border-collapse bg-white text-left">
        <caption className="sr-only">Gewichtung der Verstöße nach Schwere</caption>
        <thead>
          <tr className="border-b border-slate-300">
            <th scope="col" className="px-3 py-2">
              Schwere
            </th>
            <th scope="col" className="px-3 py-2">
              Gewicht
            </th>
          </tr>
        </thead>
        <tbody>
          {[
            ['kritisch', 10],
            ['schwer', 6],
            ['mittel', 3],
            ['gering', 1],
          ].map(([label, weight]) => (
            <tr key={label} className="border-b border-slate-200">
              <th scope="row" className="px-3 py-2 font-normal">
                {label}
              </th>
              <td className="px-3 py-2">{weight}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p className="mt-4">
        Die Zahl der betroffenen Elemente geht logarithmisch ein: fünfzig gleichartige Fehler haben
        meist dieselbe Ursache und wiegen nicht fünfzigmal so schwer wie einer. Die Summe wird auf
        die Seitengröße normiert, damit eine lange Seite nicht allein wegen ihrer Länge schlechter
        abschneidet, und über eine Exponentialkurve auf 0 bis 100 abgebildet.
      </p>
      <p className="mt-4">
        Der Wert einer Behörde ist das gewichtete Mittel ihrer Seiten: die Startseite zählt
        dreifach, die rechtlich besonders relevanten Seiten doppelt. Seiten, die nicht geladen
        werden konnten, fließen nicht ein — eine nicht erreichbare Seite ist kein Befund über
        Barrierefreiheit.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Noten</h2>
      <p className="mt-2">
        A ab 90 Punkten, B ab 80, C ab 70, D ab 60, E ab 50, darunter F.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Was der Score nicht kann</h2>
      <p className="mt-2">
        Automatisierte Tests erfassen je nach Quelle nur etwa 30 bis 40 Prozent der WCAG-Kriterien.
        Ob eine Alternativbeschreibung das Bild tatsächlich beschreibt, ob eine Überschrift den
        Abschnitt trifft, ob eine Bedienung mit der Tastatur sinnvoll durchführbar ist — das kann
        kein Programm beurteilen. Eine gute Note bedeutet: die maschinell prüfbaren Fehler sind
        selten. Sie bedeutet nicht, dass die Seite barrierefrei ist. Eine schlechte Note dagegen ist
        belastbar: die gefundenen Verstöße sind da.
      </p>
      <p className="mt-4">
        Auch der Zeitpunkt zählt: Ein Score beschreibt den Stand des letzten Scans, nicht den von
        heute.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Einwilligungsabfragen</h2>
      <p className="mt-2">
        Ein Cookie-Banner verdeckt die Seite — auch für die Prüfung. Deshalb schließen wir es
        vorher, und zwar so, wie es eine vorsichtige Besucherin täte: Wir lehnen ab. Eine Behörden-
        Website muss ohne Einwilligung nutzbar sein, und Ablehnen setzt nichts. Nur wenn es keine
        Möglichkeit zum Ablehnen gibt, klicken wir auf Zustimmen — sonst käme niemand an die Seite.
      </p>
      <p className="mt-4">
        Lässt sich die Abfrage gar nicht schließen, steht das im Ergebnis. Der Score beschreibt dann
        das Banner und nicht die Seite dahinter, und die Detailansicht sagt das ausdrücklich. Ein
        sauberes Banner ist keine saubere Website.
      </p>

      <h2 className="mt-8 text-xl font-semibold">Rücksicht beim Prüfen</h2>
      <p className="mt-2">
        Wir befolgen <code>robots.txt</code> samt Crawl-Delay und fragen jede Domain mit höchstens
        einer Seite pro Sekunde ab. Geprüft wird nur, was öffentlich erreichbar ist: keine
        Anmeldungen, keine abgeschickten Formulare, keine personenbezogenen Daten. Gespeichert
        werden die geprüften Adressen, die gefundenen Verstöße mit einem kurzen Ausschnitt des
        betroffenen HTML und die daraus errechneten Zahlen.
      </p>
    </div>
  )
}
