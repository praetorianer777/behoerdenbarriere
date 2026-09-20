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
| `lighthouse` | Node, Lighthouse | Googles Score als Vergleichswert |
| `frontend` | React, Tailwind | Ranking, Detail, Dashboard, Methodik, Statistik |
| Postgres | | Behörden, Scans, Seiten, Verstöße, Job-Queue, Nutzungszähler |

## Loslegen

```sh
cp .env.example .env
make dev        # Postgres, Chrome, API, Worker und Oberfläche starten
make seed       # Behördenliste einspielen
make import     # Landkreise aus Wikidata ergänzen
make test       # Tests
make scan URL=https://www.bund.de   # eine einzelne Seite prüfen
```

Danach liegt die Oberfläche auf <http://localhost:5173> und die API auf
<http://localhost:8080>. Wer nur Container hat und kein Go, spielt die Behördenliste
über das API-Image ein:

```sh
docker compose run --rm --entrypoint /seed api
```

Die Tests des Datenbankpakets laufen gegen eine echte Postgres-Instanz und werden ohne
`TEST_DATABASE_URL` übersprungen:

```sh
docker compose up -d postgres
TEST_DATABASE_URL='postgres://behoerdenbarriere:behoerdenbarriere@localhost:5432/behoerdenbarriere?sslmode=disable' \
  go test ./internal/store/
```

## Observability (OpenTelemetry)

Traces, metrics and trace-aware logs are exported over OTLP. **Everything is off until
`OTEL_EXPORTER_OTLP_ENDPOINT` is set**: without it no provider is installed, so a local
run and the test suite need no collector.

| | What is recorded |
| --- | --- |
| Traces (API) | one span per request, named after the chi route, with method, status and duration; `/healthz` and `/readyz` are left out |
| Traces (worker) | one `scan` span per authority with `crawl`, `page scan` and `score` beneath it, so a slow authority can be told from a slow scanner |
| Traces (database) | every query, through `otelpgx` |
| Metrics | `scans.total` by status, `scan.duration`, `scan.pages`, `jobs.queued`, `jobs.running`, `jobs.oldest_age`, the HTTP server metrics of `otelhttp` (status code included) and the pgx pool stats |
| Logs | `log/slog` as before, plus `trace_id` and `span_id` whenever a span is open |

Switching it on:

```sh
make dev-telemetry   # brings up the collector behind the `telemetry` compose profile
# in .env:
# OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
```

`OTEL_EXPORTER_OTLP_PROTOCOL` is `grpc` or `http/protobuf`, `OTEL_TRACES_SAMPLER_ARG`
is the sampling ratio between 0 and 1, and `OTEL_SERVICE_NAME` overrides the names the
binaries give themselves (`behoerdenbarriere-api`, `behoerdenbarriere-worker`). The
collector in [deploy/otel-collector.yaml](deploy/otel-collector.yaml) only prints what
it receives; a backend that keeps the data is added there.

## Woher die Liste kommt

Zwei Quellen. Von Hand gepflegt sind Bund, Länder und die größeren Städte — geprüft,
mit richtigem Namen und richtiger Adresse. Aus Wikidata kommen die rund 300 Landkreise,
weil niemand diese Breite von Hand aktuell hält.

Wo beide dieselbe Stelle kennen, gewinnt der geprüfte Eintrag: Der Import ergänzt, er
überschreibt nichts. Erkannt wird eine Dopplung am Slug und an der Adresse der Website,
damit dieselbe Behörde nicht zweimal im Ranking steht.

## Wie bewertet wird

Jeder Verstoß bekommt ein Gewicht nach seiner Schwere (critical 10, serious 6,
moderate 3, minor 1). Die Anzahl betroffener Elemente geht logarithmisch ein — fünfzig
gleichartige Fehler haben meist dieselbe Ursache und wiegen nicht fünfzigmal so schwer.
Die Summe wird auf die Seitengröße normiert und über eine Exponentialkurve auf 0 bis 100
abgebildet. Der Wert einer Behörde ist das gewichtete Mittel ihrer Seiten: die Startseite
zählt dreifach, die rechtlich besonders relevanten Seiten — Erklärung zur
Barrierefreiheit, Kontakt, Formulare — doppelt.

**Zweite Meinung.** Zu jeder Startseite wird zusätzlich Googles Lighthouse-Score
erhoben — der einzige etablierte, offen dokumentierte Wert. Er rechnet bewusst anders:
Jede Regel besteht ganz oder gar nicht. Über die bisher geprüften Behörden liegt er im
Mittel 26 Punkte über unserem und drängt sich zwischen 84 und 100, während unsere Werte
von 28 bis 100 streuen. Für eine Rangfolge taugt er deshalb kaum, als Gegenprobe für
unsere Gewichte schon.

**Grenzen des Verfahrens.** Automatisierte Prüfungen decken je nach Quelle nur etwa
30 bis 40 Prozent der WCAG-Kriterien ab. Ob eine Alternativbeschreibung das Bild
tatsächlich beschreibt, kann kein Programm beurteilen. Der Score ist ein Indikator und
kein BITV-Prüfbericht, und eine gute Note ersetzt keine manuelle Prüfung.

## Was eine Seite an Dritte sendet

Beim Prüfen läuft jede Seite ohnehin in einem echten Browser, also wird nebenbei
festgehalten, welche fremden Hosts sie kontaktiert — und vor allem **wann**: bevor eine
Einwilligung möglich war, nach unserer Ablehnung oder, wo sich nichts ablehnen ließ,
nach der Zustimmung. Ein Zähler, der beim Seitenaufruf feuert, ist etwas anderes als
einer, dem jemand zugestimmt hat.

Gespeichert wird der beobachtete Hostname, nie unsere Einordnung: Wem die Daten am Ende
gehören, sagt ein Name nicht, und eine bessere Zuordnung darf nicht bedeuten, dass jede
Behörde neu gecrawlt werden muss. In den Score fließt davon nichts ein — das ist eine
Frage des Datenschutzes, nicht der Barrierefreiheit, und beides zu vermischen macht
beide Aussagen unbrauchbar.

## Wohin die E-Mail einer Behörde geht

Jede Domain veröffentlicht im DNS, welcher Host ihre Post entgegennimmt (MX), wer in
ihrem Namen senden darf (SPF) und was mit gefälschten Absendern geschehen soll (DMARC).
`cmd/dns` liest das für alle Behörden, der Worker frischt es bei jedem Scan auf.
Gelesen wird nur, was veröffentlicht ist — kein Portscan, keine Anfrage an einen
Mailserver.

Ein MX-Eintrag sagt, wer die Post **annimmt**, nicht wer sie liest; ein SPF-Eintrag
sagt, wer senden darf, und gerade nicht, wo die Postfächer liegen. Beides steht deshalb
in getrennten Feldern, der vollständige Eintrag wird so gespeichert, wie er
veröffentlicht ist, und neben jeder Einordnung angezeigt. Auch das zählt nicht in den
Score.

## Auf einen Server bringen

Die CI baut bei jedem grünen `main` die vier eigenen Images und veröffentlicht sie in
der GitHub Container Registry (`ghcr.io/praetorianer777/behoerdenbarriere-*`), getaggt
mit `latest` und mit dem Commit. Auf dem Server holt ein Systemd-Timer sie ab:

```sh
./deploy/update.sh
```

Die Werkzeuge für den Betrieb — Behördenliste einspielen, Landkreise importieren,
DNS-Einträge holen — liegen im API-Image, weil auf einem Server kein Go steht:

```sh
docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm --entrypoint /seed api
```

Welcher Stand dort läuft, sagt die Installation selbst:

```sh
curl -s http://127.0.0.1:8081/api/v1/version
```

Der Server zieht, GitHub schiebt nicht — damit braucht niemand von außen Zugang zu der
Maschine. Einrichtung, Rückfall auf eine ältere Fassung und was dabei mit Migrationen
zu beachten ist, steht in [deploy/BETRIEB.md](deploy/BETRIEB.md).

## Grenzen der API

Die Daten sind öffentlich und sollen auch in größeren Mengen nutzbar bleiben. Die
folgenden Grenzen gibt es aus einem einzigen Grund: Ein einzelner Client darf die Seite
nicht für alle anderen lahmlegen.

| Was | Grenze | Anmerkung |
| --- | --- | --- |
| Lesende Endpunkte (`/agencies`, `/scans/{id}`) | 120 Anfragen pro Minute, Spitze 60 | je Client |
| Statistik, Regelkatalog, Drittanbieter, E-Mail (`/stats`, `/rules`, `/thirdparties`, `/mail`) | 20 pro Minute, Spitze 10 | sie rechnen über alle Scans |
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

## Zahlen über die eigene Seite

Wie oft welche Seite aufgerufen wird, zählt die API selbst — ohne Cookie, ohne fremdes
Skript, ohne Kennung im Browser. Gespeichert werden nur Tageszähler je Seite, Behörde und
API-Endpunkt. Ein Besuch ist ein Hash aus IP-Adresse und Browserkennung, dessen Schlüssel
täglich neu gezogen und nie gespeichert wird; IP-Adressen selbst werden nirgends abgelegt.
Die Zahlen stehen offen unter `/statistik`.

## Lizenz

MIT, siehe [LICENSE](LICENSE).
