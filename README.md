# Behördenbarriere

Behörden müssen barrierefrei sein. Wir schauen nach, ob sie es sind.

Behördenbarriere crawlt die Websites deutscher Behörden, prüft jede Seite mit
[axe-core](https://github.com/dequelabs/axe-core) gegen WCAG 2.1 AA — den Maßstab, den
BITV 2.0 und EN 301 549 für öffentliche Stellen setzen — und macht aus den Befunden einen
nachvollziehbaren Score von 0 bis 100 samt Schulnote. Die Ergebnisse stehen als Ranking,
Behördendetail und bundesweites Dashboard offen zur Verfügung.

## Stand

Im Aufbau. Der Fortschritt steht in den
[Issues](https://github.com/praetorianer777/behoerdenbarriere/issues).

## Aufbau

| Teil | Technik | Aufgabe |
| --- | --- | --- |
| `backend/cmd/api` | Go, chi | REST-Schnittstelle für das Frontend |
| `backend/cmd/worker` | Go | Crawl, Prüfung, Bewertung, Zeitplan |
| `backend/internal/scanner` | chromedp, axe-core | Seite laden, prüfen, Links auslesen |
| `backend/internal/scoring` | Go | Befunde in Score und Note umrechnen |
| `frontend` | React, Tailwind | Ranking, Detail, Dashboard, Methodik, Statistik |
| Postgres | | Behörden, Scans, Seiten, Verstöße, Job-Queue, Nutzungszähler |

## Loslegen

```sh
cp .env.example .env
make dev        # Postgres, Chrome, API und Worker starten
make test       # Tests
make scan URL=https://www.bund.de   # eine einzelne Seite prüfen
```

Die Tests des Datenbankpakets laufen gegen eine echte Postgres-Instanz und werden ohne
`TEST_DATABASE_URL` übersprungen:

```sh
docker compose up -d postgres
TEST_DATABASE_URL='postgres://behoerdenbarriere:behoerdenbarriere@localhost:5432/behoerdenbarriere?sslmode=disable' \
  go test ./internal/store/
```

## Wie bewertet wird

Jeder Verstoß bekommt ein Gewicht nach seiner Schwere (critical 10, serious 6,
moderate 3, minor 1). Die Anzahl betroffener Elemente geht logarithmisch ein — fünfzig
gleichartige Fehler haben meist dieselbe Ursache und wiegen nicht fünfzigmal so schwer.
Die Summe wird auf die Seitengröße normiert und über eine Exponentialkurve auf 0 bis 100
abgebildet. Der Wert einer Behörde ist das gewichtete Mittel ihrer Seiten: die Startseite
zählt dreifach, die rechtlich besonders relevanten Seiten — Erklärung zur
Barrierefreiheit, Kontakt, Formulare — doppelt.

**Grenzen des Verfahrens.** Automatisierte Prüfungen decken je nach Quelle nur etwa
30 bis 40 Prozent der WCAG-Kriterien ab. Ob eine Alternativbeschreibung das Bild
tatsächlich beschreibt, kann kein Programm beurteilen. Der Score ist ein Indikator und
kein BITV-Prüfbericht, und eine gute Note ersetzt keine manuelle Prüfung.

## Rücksicht beim Crawlen

`robots.txt` und Crawl-Delay werden befolgt, jede Domain wird mit höchstens einer
Anfrage pro Sekunde abgefragt, der User-Agent nennt das Projekt. Geprüft wird nur, was
öffentlich erreichbar ist: keine Anmeldungen, keine abgeschickten Formulare, keine
personenbezogenen Daten. Wer seine Seiten nicht im Ranking sehen möchte, kann sie
abschalten lassen.

## Zahlen über die eigene Seite

Wie oft welche Seite aufgerufen wird, zählt die API selbst — ohne Cookie, ohne fremdes
Skript, ohne Kennung im Browser. Gespeichert werden nur Tageszähler je Seite, Behörde und
API-Endpunkt. Ein Besuch ist ein Hash aus IP-Adresse und Browserkennung, dessen Schlüssel
täglich neu gezogen und nie gespeichert wird; IP-Adressen selbst werden nirgends abgelegt.
Die Zahlen stehen offen unter `/statistik`.

## Lizenz

MIT, siehe [LICENSE](LICENSE).
