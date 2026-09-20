#!/bin/sh
# Tägliche Sicherung der Datenbank.
#
# Die Scan-Historie ist der einzige Teil, den ein neuer Lauf nicht wiederherstellen
# kann: Ein Scan misst den heutigen Stand, nicht den von letztem Jahr. Alles andere —
# Behördenliste, Ergebnisse, Befunde — entsteht beim nächsten Durchlauf neu.
set -eu

keep_days="${BACKUP_KEEP_DAYS:-14}"

while true; do
    stamp="$(date -u +%Y-%m-%dT%H-%M-%SZ)"
    ziel="/backups/behoerdenbarriere-${stamp}.sql.gz"

    if pg_dump --no-owner --no-privileges | gzip > "${ziel}.teil"; then
        mv "${ziel}.teil" "$ziel"
        echo "$(date -u +%FT%TZ) Sicherung geschrieben: $ziel ($(du -h "$ziel" | cut -f1))"
    else
        # Eine fehlgeschlagene Sicherung darf keine halbe Datei hinterlassen, die
        # später wie eine gültige aussieht.
        rm -f "${ziel}.teil"
        echo "$(date -u +%FT%TZ) Sicherung fehlgeschlagen" >&2
    fi

    find /backups -name 'behoerdenbarriere-*.sql.gz' -mtime "+${keep_days}" -delete
    sleep 86400
done
