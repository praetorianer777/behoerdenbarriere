# Betrieb

Für einen eigenen Server mit Docker und einem Nginx Proxy Manager davor.

## Einmalig einrichten

```sh
git clone https://github.com/praetorianer777/behoerdenbarriere.git
cd behoerdenbarriere
cp .env.example .env
```

In der `.env` müssen gesetzt werden:

| Variable | Bedeutung |
| --- | --- |
| `POSTGRES_PASSWORD` | Passwort der Datenbank. Ohne es startet nichts. |
| `API_KEY` | Schlüssel für den manuellen Rescan und die höhere Abrufgrenze. |
| `PUBLIC_URL` | Die öffentliche Adresse, z. B. `https://behoerdenbarriere.de`. Sie steuert die CORS-Freigabe. |
| `PROXY_NETWORK` | Name des Docker-Netzes des Nginx Proxy Managers, Vorgabe `npm`. |

Die Angaben zum Betreiber stehen in `frontend/src/betreiber.ts` und gehören ins
Impressum und in die Datenschutzerklärung. **Solange dort Platzhalter stehen, weist die
Website sichtbar darauf hin.** Das ist Absicht: Ein erfundenes Impressum wäre schlimmer
als ein fehlendes.

## Starten

```sh
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm --entrypoint /seed api
docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm --entrypoint /import api
```

Der erste Befehl startet alles, der zweite spielt die von Hand gepflegte Behördenliste
ein, der dritte ergänzt die Landkreise aus Wikidata. Danach arbeitet der Worker die
Warteschlange von selbst ab und nimmt jede Behörde wieder auf, deren letzte Prüfung
älter als `RESCAN_INTERVAL` ist.

Kein Dienst veröffentlicht einen Port auf dem Host. Die Oberfläche hängt zusätzlich im
Netz des Proxys und ist dort unter ihrem Containernamen erreichbar.

## Nginx Proxy Manager

Neuer Proxy Host:

- **Domain Names:** die öffentliche Adresse
- **Scheme:** `http`
- **Forward Hostname / IP:** `behoerdenbarriere-frontend-1` (oder wie der Container bei
  Ihnen heißt — `docker ps` zeigt es)
- **Forward Port:** `80`
- **Websockets Support:** aus, wird nicht gebraucht
- **SSL:** Zertifikat anfordern, *Force SSL* und *HTTP/2* einschalten

Der Container muss im selben Docker-Netz liegen wie der Proxy Manager; genau dafür ist
`PROXY_NETWORK` da. Die API wird nicht getrennt veröffentlicht: Die Oberfläche reicht
`/api` intern an sie weiter.

Der Proxy setzt `X-Forwarded-For`. Die API glaubt diesen Kopf nur, wenn die Anfrage aus
einem der Netze in `API_TRUSTED_PROXIES` kommt — voreingestellt sind Loopback und die
privaten Bereiche, was für Docker passt. Steht der Proxy woanders, muss dessen Netz
dort eingetragen werden, sonst landen alle Aufrufer in einem gemeinsamen Zähler.

## Sicherungen

Der Dienst `backup` schreibt täglich einen `pg_dump` in das Volume `backups` und
räumt alles auf, was älter als `BACKUP_KEEP_DAYS` (Vorgabe 14) ist.

Die Scan-Historie ist der einzige Teil, den ein neuer Lauf nicht wiederherstellen kann:
Ein Scan misst den heutigen Stand, nicht den von letztem Jahr. Alles andere entsteht
beim nächsten Durchlauf neu.

Einspielen einer Sicherung:

```sh
docker compose -f docker-compose.yml -f docker-compose.prod.yml stop api worker
gunzip -c /var/lib/docker/volumes/behoerdenbarriere_backups/_data/behoerdenbarriere-….sql.gz \
  | docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T postgres psql -U behoerdenbarriere
docker compose -f docker-compose.yml -f docker-compose.prod.yml start api worker
```

Es lohnt, das einmal auszuprobieren, bevor man es braucht — und die Sicherungen vom
Server herunterzuholen. Eine Sicherung, die auf derselben Platte liegt wie das Original,
ist bei einem Plattenschaden mit weg.

## Aktualisieren

```sh
git pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

Migrationen laufen beim Start von API und Worker von selbst mit.

## Nachsehen, was los ist

```sh
docker compose -f docker-compose.yml -f docker-compose.prod.yml logs -f worker
curl -s https://…/api/v1/stats | jq '{agencies, scanned, avg_score}'
```

Wer es genauer braucht: Mit `OTEL_EXPORTER_OTLP_ENDPOINT` schicken API und Worker
Traces und Metriken an einen Collector; `docker compose --profile telemetry up -d`
startet einen mit.

## Last bei den Behörden

Voreingestellt sind eine Seite pro Sekunde und Behörde und 60 Seiten je Prüfung, und
jede Behörde wird einmal pro Woche geprüft. `robots.txt` wird befolgt, ein
Crawl-Delay darin hat Vorrang vor unserer eigenen Rate.

Wer die Liste vergrößert, sollte `CRAWL_RATE_PER_SEC` und `CRAWL_MAX_PAGES` im Blick
behalten: Am Ende steht hinter jeder Zeile im Ranking eine Behörde, deren Server wir
beschäftigen.
