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
