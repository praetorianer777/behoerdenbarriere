#!/bin/sh
# Prüft, dass ein schlichtes "docker compose ..." in einer Installation die
# Betriebsfassung nimmt — und dass das Weglassen der Zeile wirklich schadet.
#
# Warum es das gibt: Die Anleitung hatte vor jedem Befehl neun Wörter Präfix stehen.
# Einmal vergessen, und Compose baute Images aus dem Quelltext, veröffentlichte den
# Datenbank-Port auf dem Host und hielt die Sicherung für einen Überrest. Nichts davon
# scheitert laut, also muss eine Prüfung danach sehen.
set -eu

cd "$(dirname "$0")/.."

arbeit=$(mktemp -d)
trap 'rm -rf "$arbeit"' EXIT

# Eine Installation, wie die Anleitung sie beschreibt: das Verzeichnis samt .env, in
# dem der Operator dann seine Befehle eintippt.
cp docker-compose.yml docker-compose.prod.yml "$arbeit/"
cp .env.example "$arbeit/.env"
mkdir -p "$arbeit/deploy"
cp deploy/backup.sh "$arbeit/deploy/"
{
    echo "POSTGRES_PASSWORD=pruefung"
    echo "API_KEY=pruefung"
    echo "PUBLIC_URL=https://example.org"
} >>"$arbeit/.env"

aufloesen() {
    # Ohne geerbte Umgebung: Geprüft wird, was die .env allein bewirkt.
    env -i PATH="$PATH" HOME="$HOME" docker compose --project-directory "$arbeit" config
}

ohne=$(aufloesen)
if ! printf '%s' "$ohne" | grep -q 'published: "5432"'; then
    echo "Erwartet war, dass die Entwicklungsfassung den Datenbank-Port veröffentlicht — sie tut es nicht mehr." >&2
    echo "Dann prüft dieses Skript nichts. Anlass nachsehen, statt die Prüfung zu streichen." >&2
    exit 1
fi

sed -i 's|^# COMPOSE_FILE=|COMPOSE_FILE=|' "$arbeit/.env"
mit=$(aufloesen)

fehler=0
meckern() {
    echo "$1" >&2
    fehler=1
}

printf '%s' "$mit" | grep -q 'published: "5432"' &&
    meckern "Der Datenbank-Port wird auf dem Host veröffentlicht."
printf '%s' "$mit" | grep -q 'published: "9222"' &&
    meckern "Der Chrome-Port wird auf dem Host veröffentlicht."
printf '%s' "$mit" | grep -q 'published: "8081"' ||
    meckern "Der Port der Oberfläche fehlt."
printf '%s' "$mit" | grep -q 'ghcr.io/praetorianer777/behoerdenbarriere-api' ||
    meckern "Die API käme nicht aus der Registry."
printf '%s' "$mit" | grep -q 'backup:' ||
    meckern "Die Sicherung fehlt und gälte als Überrest."

if [ "$fehler" -ne 0 ]; then
    echo >&2
    echo "Ein schlichtes 'docker compose' in einer Installation nimmt nicht die Betriebsfassung." >&2
    exit 1
fi

echo "Kurzbefehl in Ordnung: 'docker compose' löst in einer Installation auf die Betriebsfassung auf."
