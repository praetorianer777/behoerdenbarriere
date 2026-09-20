#!/bin/sh
# Prüft, dass jedes Werkzeug, zu dem die Betriebsanleitung auffordert, im Image auch
# vorhanden ist.
#
# Anleitung und Dockerfile stehen in verschiedenen Dateien und sind genau deshalb
# auseinandergelaufen: `--entrypoint /import` stand in der Anleitung, im Image lag es
# nie. Auf dem Server endet das mit „no such file or directory" — nach dem Pull, nach
# der Einrichtung, beim ersten Befehl, der zählt.
set -eu

cd "$(dirname "$0")/.."

image=${1:-}
if [ -z "$image" ]; then
    image=behoerdenbarriere-api:pruefung
    docker build --target api -t "$image" ./backend >/dev/null
fi

# Das Image hat keine Shell — distroless. Also wird sein Inhalt ausgepackt und gelesen.
container=$(docker create "$image")
trap 'docker rm -f "$container" >/dev/null 2>&1 || true' EXIT
contents=$(docker export "$container" | tar -t)

missing=0
for entrypoint in $(grep -ho -- '--entrypoint /[a-z]*' deploy/BETRIEB.md README.md | awk '{print $2}' | sort -u); do
    if printf '%s\n' "$contents" | grep -qx "${entrypoint#/}"; then
        echo "  $entrypoint ist im Image"
    else
        # Der Fall, für den es diese Prüfung gibt.
        echo "Die Anleitung nennt '$entrypoint', das Image enthält es nicht." >&2
        missing=1
    fi
done

exit "$missing"
