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
| `PROXY_NETWORK` | Name des Docker-Netzes des Nginx Proxy Managers, Vorgabe `npm`. **Der Name ist fast nie `npm`** — siehe [Das Netz des Proxys](#das-netz-des-proxys). |

Die Angaben zum Betreiber stehen in `frontend/src/betreiber.ts` und gehören ins
Impressum und in die Datenschutzerklärung. **Solange dort Platzhalter stehen, weist die
Website sichtbar darauf hin.** Das ist Absicht: Ein erfundenes Impressum wäre schlimmer
als ein fehlendes.

## Das Netz des Proxys

Die Oberfläche hängt im selben Docker-Netz wie der Nginx Proxy Manager, sonst kommt der
Proxy nicht an sie heran. Dieses Netz gehört ihm, nicht uns — deshalb ist es in der
Betriebsfassung als `external` eingetragen, und deshalb bricht der Start ab, wenn der
Name nicht stimmt:

```
network npm declared as external, but could not be found
```

Dann startet **gar nichts**. Der richtige Name steht in:

```sh
docker network ls
```

Ein per Compose installierter Proxy Manager nennt sein Netz nach seinem Projekt, meist
`nginxproxymanager_default`. Diesen Namen in die `.env`:

```sh
echo "PROXY_NETWORK=nginxproxymanager_default" >> .env
```

Wer es sauberer trennen will, legt ein eigenes Netz an und hängt den Proxy zusätzlich
hinein — dann bleibt der Name stabil, auch wenn der Proxy neu aufgesetzt wird:

```sh
docker network create npm
docker network connect npm <container-des-proxys>
```

## Starten

```sh
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm --entrypoint /seed api
docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm --entrypoint /import api
```

Beim allerersten Mal muss die Registry einmal von Hand freigegeben werden: GitHub legt
neue Pakete **privat** an. Unter `github.com/users/praetorianer777/packages` bei jedem
der vier Pakete unter *Package settings* die Sichtbarkeit auf *public* stellen — danach
zieht der Server ohne Anmeldung. Ob es klappt, sagt:

```sh
docker pull ghcr.io/praetorianer777/behoerdenbarriere-api:latest
```

Wer die Pakete privat lassen will, meldet den Server stattdessen einmalig mit einem
Token an (`docker login ghcr.io`); dann liegt allerdings ein Token auf der Maschine, das
irgendwann abläuft.

Gebaut wird dabei nichts: Die vier eigenen Images (`api`, `worker`, `frontend`,
`lighthouse`) kommen fertig aus der CI und liegen öffentlich in der GitHub Container
Registry. Veröffentlicht wird nur, was die vollständige Prüfung bestanden hat — `latest`
zeigt also immer auf einen Stand, der grün war.

Der erste Befehl startet alles, der zweite spielt die von Hand gepflegte Behördenliste
ein (rund 130 Einträge), der dritte ergänzt die Landkreise aus Wikidata (rund 290).
**Ohne den dritten fehlt die Ebene, auf der die meisten Menschen tatsächlich mit einer
Behörde zu tun haben.** Danach arbeitet der Worker die
Warteschlange von selbst ab und nimmt jede Behörde wieder auf, deren letzte Prüfung
älter als `RESCAN_INTERVAL` ist.

Die DNS-Einträge aller Behörden lassen sich einmal am Stück holen; im laufenden Betrieb
frischt der Worker sie bei jeder Prüfung mit auf:

```sh
docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm --entrypoint /dns api
```

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

Von Hand, sofort:

```sh
./deploy/update.sh
```

Das holt die zuletzt veröffentlichten Images, startet neu, was sich geändert hat, und
räumt die abgelösten weg. Migrationen laufen beim Start von API und Worker von selbst
mit.

Automatisch, täglich:

```sh
sudo cp deploy/behoerdenbarriere-update.service deploy/behoerdenbarriere-update.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now behoerdenbarriere-update.timer
systemctl list-timers behoerdenbarriere-update
```

Die Einheiten erwarten das Projekt unter `/opt/behoerdenbarriere`; liegt es woanders,
sind `WorkingDirectory` und `ExecStart` in der `.service` anzupassen.

**Der Server holt, GitHub schiebt nicht.** Deshalb braucht GitHub keinen Zugang zu
dieser Maschine: kein Schlüssel als Secret, kein von außen erreichbarer SSH-Port, nichts
zu widerrufen, wenn irgendwo ein Token ausläuft. Der Preis ist, dass eine Änderung erst
mit dem nächsten Lauf ankommt — oder eben mit `deploy/update.sh` von Hand.

Schlägt ein Update fehl, endet der Dienst mit einem Fehler und die laufenden Container
bleiben, wie sie sind. `journalctl -u behoerdenbarriere-update` sagt, woran es lag.

### Auf eine bestimmte Fassung zurück

Jedes Image trägt neben `latest` auch den Commit, aus dem es gebaut wurde:

```sh
echo "IMAGE_TAG=6f2c1ab…" >> .env
./deploy/update.sh
```

**Über eine Migration hinweg ist das nicht damit getan.** Migrationen laufen nur
vorwärts: Eine Datenbank, die der neue Stand bereits umgebaut hat, versteht der alte
unter Umständen nicht mehr. Wenn zwischen den beiden Fassungen eine neue Datei in
`backend/internal/store/migrations/` liegt, gehört zum Zurück auch das Einspielen der
letzten Sicherung — siehe [Sicherungen](#sicherungen). Ohne Migration dazwischen genügt
der Tag.

Wieder nach vorn: die Zeile `IMAGE_TAG` aus der `.env` entfernen und `deploy/update.sh`
noch einmal.

## Vorher ausprobieren

Wer eine Änderung an der Betriebsfassung, am Image oder an dieser Anleitung macht, kann
die Installation auf dem eigenen Rechner nachspielen — dieselbe Prüfung läuft in der CI:

```sh
./deploy/check-installation.sh
```

Sie baut die Images aus dem aktuellen Stand, startet die Betriebsfassung, spielt die
Behördenliste ein und fragt die Seite und die API durch das nginx der Oberfläche ab.
Danach räumt sie alles wieder weg.

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
