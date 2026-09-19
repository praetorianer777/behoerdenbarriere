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
| `frontend` | React, Tailwind | Ranking, Detail, Dashboard, Methodik |
| Postgres | | Behörden, Scans, Seiten, Verstöße, Job-Queue |

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

## Grenzen der API

Die Daten sind öffentlich und sollen auch in größeren Mengen nutzbar bleiben. Die
folgenden Grenzen gibt es aus einem einzigen Grund: Ein einzelner Client darf die Seite
nicht für alle anderen lahmlegen.

| Was | Grenze | Anmerkung |
| --- | --- | --- |
| Lesende Endpunkte (`/agencies`, `/scans/{id}`) | 120 Anfragen pro Minute, Spitze 60 | je Client |
| Statistik und Regelkatalog (`/stats`, `/rules`) | 20 pro Minute, Spitze 10 | sie rechnen über alle Scans |
| Mit API-Schlüssel (`X-API-Key`) | 600 pro Minute, Spitze 200 | lieber einen Schlüssel erfragen, als die Grenze zu umgehen |
| `POST /agencies/{slug}/rescan` | Schlüssel nötig, dazu ein Rescan je Behörde pro Stunde | die Last landet bei der Behörde |
| Anfragekörper | 64 KiB | |
| Verlaufspunkte je Behörde | 200 | längere Verläufe werden gekürzt |
| Einträge je Liste in einer Antwort | 500 | die Seitengröße ist getrennt auf 200 begrenzt |

Eine abgewiesene Anfrage bekommt `429` mit `Retry-After` und `X-RateLimit-Limit`,
`X-RateLimit-Remaining` sowie `X-RateLimit-Reset` (in Sekunden, `Reset` zählt bis zum
vollen Eimer). Jede lesende Antwort trägt `Cache-Control: public, max-age=300` und ein
`ETag`; wer es als `If-None-Match` zurückschickt, bekommt `304` und spart beiden Seiten
die Arbeit — ein Scan-Ergebnis ändert sich höchstens einmal pro Woche.

Clients werden über die IP-Adresse unterschieden, oder über den API-Schlüssel, wenn
einer mitgeschickt wird. Hinter einem Proxy stammt die Adresse aus `X-Forwarded-For`,
aber nur, wenn die Anfrage tatsächlich aus einem Netz in `API_TRUSTED_PROXIES` kam
(voreingestellt Loopback und die privaten Bereiche, wie im Compose-Setup; `none` schaltet
es ab). Diesen Kopf kann jeder schreiben — von einem nicht vertrauenswürdigen Gegenüber
wird er deshalb ignoriert, sonst könnte sich ein Client für jede Anfrage eine neue
Identität ausdenken und die Grenzen wären wertlos.

Alle Grenzen sind konfigurierbar, siehe `.env.example`; eine Null schaltet eine ab.

## Rücksicht beim Crawlen

`robots.txt` und Crawl-Delay werden befolgt, jede Domain wird mit höchstens einer
Anfrage pro Sekunde abgefragt, der User-Agent nennt das Projekt. Geprüft wird nur, was
öffentlich erreichbar ist: keine Anmeldungen, keine abgeschickten Formulare, keine
personenbezogenen Daten. Wer seine Seiten nicht im Ranking sehen möchte, kann sie
abschalten lassen.

## Lizenz

MIT, siehe [LICENSE](LICENSE).
