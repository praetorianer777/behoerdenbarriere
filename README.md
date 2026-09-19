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

## API limits

The data is public and meant to be used in bulk. The limits below exist for one reason
only: a single client must not be able to take the site down for everyone.

| What | Limit | Notes |
| --- | --- | --- |
| Read endpoints (`/agencies`, `/scans/{id}`) | 120 requests per minute, burst 60 | per client |
| Statistics and rule catalogue (`/stats`, `/rules`) | 20 per minute, burst 10 | they aggregate over every scan |
| With an API key (`X-API-Key`) | 600 per minute, burst 200 | ask for a key instead of scraping around the limit |
| `POST /agencies/{slug}/rescan` | key required, plus one rescan per authority per hour | the traffic lands on that authority |
| Request body | 64 KiB | |
| History points per authority | 200 | longer histories are truncated |
| Items per list in one response | 500 | the page size is capped at 200 separately |

A rejected request answers `429` with `Retry-After` and `X-RateLimit-Limit`,
`X-RateLimit-Remaining` and `X-RateLimit-Reset` (all seconds, `Reset` counts to a full
bucket). Every read answer carries `Cache-Control: public, max-age=300` and an `ETag`;
sending it back as `If-None-Match` gets a `304` and costs neither side anything, which
is worth doing — a scan result changes at most once a week.

Clients are told apart by IP address, or by API key if one is presented. Behind a proxy
the address is taken from `X-Forwarded-For`, but only when the request actually came
from a network listed in `API_TRUSTED_PROXIES` (by default loopback and the private
ranges, which is what the compose setup uses; `none` disables it). Anyone can write that
header, so from an untrusted peer it is ignored — otherwise a client could invent a new
identity per request and the limits would mean nothing.

Every limit is configurable, see `.env.example`; a zero switches one off.

## Rücksicht beim Crawlen

`robots.txt` und Crawl-Delay werden befolgt, jede Domain wird mit höchstens einer
Anfrage pro Sekunde abgefragt, der User-Agent nennt das Projekt. Geprüft wird nur, was
öffentlich erreichbar ist: keine Anmeldungen, keine abgeschickten Formulare, keine
personenbezogenen Daten. Wer seine Seiten nicht im Ranking sehen möchte, kann sie
abschalten lassen.

## Lizenz

MIT, siehe [LICENSE](LICENSE).
