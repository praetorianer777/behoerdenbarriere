#!/bin/sh
# Holt die zuletzt veröffentlichte Fassung und startet neu, was sich geändert hat.
#
# Der Server zieht, GitHub schiebt nicht: Damit braucht niemand von außen Zugang zu
# dieser Maschine — kein Schlüssel als Secret, kein offener Port, nichts zu widerrufen.
#
# Migrationen laufen beim Start der API. Ein Rückfall auf eine ältere Fassung ist
# deshalb nicht einfach „alten Tag setzen", sobald eine Migration dazwischenliegt;
# deploy/BETRIEB.md sagt, was dann zu tun ist.
set -eu

cd "$(dirname "$0")/.."
compose="docker compose -f docker-compose.yml -f docker-compose.prod.yml"

# Ohne neue Images ist nichts zu tun — und ein Abbruch hier lässt den laufenden Stand
# unangetastet, statt ihn halb zu ersetzen.
$compose pull --quiet

$compose up -d

# Die abgelösten Images belegen sonst mit jeder Veröffentlichung mehr Platz. Nur was
# kein Container mehr braucht, und nichts, was noch mit einem Namen versehen ist.
docker image prune --force >/dev/null

$compose ps

# Was jetzt wirklich läuft. Ein Tag wie "latest" sagt darüber nichts: Ein Start ohne
# vorheriges Holen erzeugt neue Container aus der alten Kopie und sieht dabei genauso
# aus wie ein erfolgreiches Update.
running=$(curl --silent --fail --max-time 5 --retry 10 --retry-delay 2 --retry-all-errors \
    "http://127.0.0.1:${WEB_PORT:-8081}/api/v1/version" 2>/dev/null || true)
if [ -n "$running" ]; then
    echo
    echo "Es läuft: $running"
fi
