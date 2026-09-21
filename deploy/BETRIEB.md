# Betrieb

Für einen eigenen Server mit Docker und einem Nginx Proxy Manager davor.

## Einmalig einrichten

```sh
git clone https://github.com/praetorianer777/behoerdenbarriere.git
cd behoerdenbarriere
cp .env.example .env
sed -i 's|^# COMPOSE_FILE=|COMPOSE_FILE=|' .env
```

Die zweite Zeile ist keine Kosmetik. Ohne `COMPOSE_FILE` nimmt ein schlichtes
`docker compose ...` in diesem Verzeichnis die Entwicklungsfassung: Es baut die Images
aus dem Quelltext, statt die geholten zu nehmen, **veröffentlicht den Datenbank-Port auf
dem Host** und hält die Sicherung für einen Überrest, den man wegräumen könnte. Nichts
davon scheitert laut. Mit der Zeile ist der kurze Befehl der richtige.

In der `.env` müssen gesetzt werden:

| Variable | Bedeutung |
| --- | --- |
| `POSTGRES_PASSWORD` | Passwort der Datenbank. Ohne es startet nichts. |
| `API_KEY` | Schlüssel für den manuellen Rescan und die höhere Abrufgrenze. |
| `PUBLIC_URL` | Die öffentliche Adresse, z. B. `https://behoerdenbarriere.de`. Sie steuert die CORS-Freigabe. |
| `WEB_PORT` | Port auf dem Server, auf dem die Oberfläche erscheint. Vorgabe `8081`. Darauf wird der Proxy gerichtet. |
| `WEB_BIND` | Adresse, auf der dieser Port erscheint. Vorgabe `0.0.0.0`, also überall — siehe [Der Port nach außen](#der-port-nach-außen). |
| `COMPOSE_FILE` | `docker-compose.yml:docker-compose.prod.yml`. Macht die Betriebsfassung zur Vorgabe, siehe oben. |
| `OPERATOR_NAME`, `OPERATOR_STREET`, `OPERATOR_CITY`, `OPERATOR_EMAIL`, `OPERATOR_HOSTING` | Wer die Seite betreibt und wo sie läuft — Impressum und Datenschutzerklärung. Pflicht nach § 5 DDG. Solange etwas fehlt, weist die Website sichtbar darauf hin. |
| `OPERATOR_COUNTRY`, `OPERATOR_PHONE`, `OPERATOR_VAT_ID` | Land (Vorgabe Deutschland), Telefon und USt-IdNr. (freiwillig). |

Die Angaben zum Betreiber kommen aus der `.env` und nicht aus dem Quelltext: Wer die
veröffentlichten Images nutzt, kann den Code nicht anfassen — und soll es für sein
eigenes Impressum auch nicht müssen. **Solange Angaben fehlen, weist die Website
sichtbar darauf hin.** Das ist Absicht: Ein erfundenes Impressum wäre schlimmer als ein
fehlendes.

## Der Port nach außen

Nach außen zeigt genau ein Port: der der Oberfläche, voreingestellt `8081`. Auf den wird
der Proxy gerichtet. Die API bekommt bewusst keinen eigenen — das nginx der Oberfläche
reicht `/api` intern an sie weiter, ein zweiter Port wäre ein zweiter Weg hinein, ohne
dass man etwas davon hätte.

Ein veröffentlichter Port ist für jeden erreichbar, der die Maschine erreicht. Steht der
Proxy auf einer festen Adresse, gehört der Port auf sie beschränkt:

```sh
sudo ufw allow from 192.168.1.10 to any port 8081 proto tcp
```

Wer stattdessen ein privates Netz zwischen beiden Maschinen hat, kann den Port auch nur
dort erscheinen lassen:

```sh
echo "WEB_BIND=10.8.0.3" >> .env
```

**Docker umgeht die Firewall.** Ein veröffentlichter Port hängt seine eigenen Regeln vor
die von `ufw`; `ufw deny` allein reicht dafür nicht. Entweder `WEB_BIND` auf eine
Adresse setzen, die von außen gar nicht erreichbar ist, oder die Regel in der
`DOCKER-USER`-Kette anlegen:

```sh
sudo iptables -I DOCKER-USER -p tcp --dport 8081 ! -s 192.168.1.10 -j DROP
```

## Starten

```sh
./deploy/update.sh
docker compose run --rm --entrypoint /seed api
docker compose run --rm --entrypoint /import api
```

Gestartet wird mit demselben Skript, mit dem später aktualisiert wird — **nicht** mit
einem blanken `docker compose up -d`. Das holt nämlich kein Image, dessen Tag lokal
schon vorhanden ist: `latest` wandert mit jeder Veröffentlichung, und ein Start ohne
vorheriges Holen baut neue Container aus der alten Kopie. Er sieht dabei aus wie ein
gelungenes Update, mit frischen Containern und allem. Das ist genau einmal passiert und
hat eine Stunde gekostet.

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
docker compose run --rm --entrypoint /dns api
```

Außer der Oberfläche veröffentlicht kein Dienst einen Port auf dem Host — Datenbank,
API, Chrome und Lighthouse sind nur für die anderen Container erreichbar.

## Nginx Proxy Manager

Neuer Proxy Host:

- **Domain Names:** die öffentliche Adresse
- **Scheme:** `http`
- **Forward Hostname / IP:** die Adresse des Servers, auf dem diese Installation läuft
- **Forward Port:** der `WEB_PORT`, voreingestellt `8081`
- **Websockets Support:** aus, wird nicht gebraucht
- **SSL:** Zertifikat anfordern, *Force SSL* und *HTTP/2* einschalten

Der Proxy darf auf einer eigenen Maschine stehen; er spricht den Server über das Netz an
wie jeder andere Aufrufer. Die API wird nicht getrennt veröffentlicht: Die Oberfläche
reicht `/api` intern an sie weiter.

Die Adresse der Besuchenden geht über zwei Stationen — Proxy, dann das nginx der
Oberfläche. Beide hängen sie an `X-Forwarded-For` an, und die API glaubt den Kopf nur
bei einer Anfrage aus einem Netz in `API_TRUSTED_PROXIES`. Voreingestellt sind Loopback
und die privaten Bereiche; für einen Proxy im eigenen Netz passt das. Steht er unter
einer öffentlichen Adresse, muss die dort eingetragen werden — sonst landen alle
Aufrufer in einem gemeinsamen Zähler.

## Sicherungen

Der Dienst `backup` schreibt täglich einen `pg_dump` in das Volume `backups` und
räumt alles auf, was älter als `BACKUP_KEEP_DAYS` (Vorgabe 14) ist.

Die Scan-Historie ist der einzige Teil, den ein neuer Lauf nicht wiederherstellen kann:
Ein Scan misst den heutigen Stand, nicht den von letztem Jahr. Alles andere entsteht
beim nächsten Durchlauf neu.

Einspielen einer Sicherung:

```sh
docker compose stop api worker
gunzip -c /var/lib/docker/volumes/behoerdenbarriere_backups/_data/behoerdenbarriere-….sql.gz \
  | docker compose exec -T postgres psql -U behoerdenbarriere
docker compose start api worker
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

## Wie viele Prüfungen gleichzeitig

`WORKER_CONCURRENCY` in der `.env`, Vorgabe 4. Eine Prüfung wartet fast die ganze Zeit
— auf den Takt gegenüber der Behörde, auf das Nachladen der Seite —, kostet also kaum
Rechenzeit. Teurer ist der Arbeitsspeicher: Jede gleichzeitige Prüfung hält einen
Browser-Tab offen, grob 50 bis 100 MB.

Für die Behörden ändert sich dadurch nichts: Der Takt von einer Anfrage pro Sekunde
gilt je Host und über alle laufenden Prüfungen hinweg. Vier gleichzeitige Prüfungen
unter `bund.de` teilen sich also eine Anfrage pro Sekunde, nicht vier.

Höher drehen lohnt sich, solange Speicher da ist:

```sh
echo "WORKER_CONCURRENCY=8" >> .env
./deploy/update.sh
docker stats --no-stream
```

## Nachsehen, was los ist

Welcher Stand läuft:

```sh
curl -s http://127.0.0.1:${WEB_PORT:-8081}/api/v1/version
```

Die Antwort nennt den Commit, aus dem das Image gebaut wurde. Zum Vergleich, was
veröffentlicht ist:

```sh
git ls-remote https://github.com/praetorianer777/behoerdenbarriere.git HEAD
```

Stimmen die nicht überein, fehlt ein `./deploy/update.sh` — oder die Veröffentlichung
läuft noch.

```sh
docker compose logs -f worker
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
