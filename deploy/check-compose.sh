#!/bin/sh
# Prüft die Betriebsfassung, ohne sie zu starten: Lässt sie sich überhaupt auflösen,
# und zeigt jedes selbst gebaute Image auf eines, das die CI auch veröffentlicht?
#
# Ein Name, der hier auseinanderläuft, fällt sonst erst auf dem Server auf — dann als
# Container, der nicht mehr hochkommt.
set -eu

cd "$(dirname "$0")/.."

# Platzhalter für die Werte, die auf dem Server in der .env stehen. Geprüft wird die
# Struktur, nicht die Einrichtung.
POSTGRES_PASSWORD=pruefung \
API_KEY=pruefung \
PUBLIC_URL=https://example.org \
  docker compose -f docker-compose.yml -f docker-compose.prod.yml config >/tmp/compose-prod.yaml

missing=0
for image in $(grep -o 'ghcr\.io/[^:]*behoerdenbarriere-[a-z]*' /tmp/compose-prod.yaml | sort -u); do
    name=${image##*behoerdenbarriere-}
    if ! grep -q "name: $name\$" .github/workflows/ci.yml; then
        echo "Die CI veröffentlicht kein Image für '$name', die Betriebsfassung erwartet es aber." >&2
        missing=1
    fi
done

if [ "$missing" -ne 0 ]; then
    exit 1
fi

echo "Betriebsfassung in Ordnung: $(grep -c 'ghcr\.io' /tmp/compose-prod.yaml) eigene Images, alle veröffentlicht."
