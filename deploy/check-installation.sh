#!/bin/sh
# Fährt eine frische Installation so hoch, wie die Betriebsanleitung es beschreibt, und
# prüft, ob dabei etwas Benutzbares herauskommt.
#
# Warum es das gibt: Die übrigen Prüfungen testen das Produkt. Die e2e-Tests fahren das
# Frontend gegen eine gestubbte API, der Image-Job hat gebaut und nie etwas gestartet.
# Der Weg, den die Anleitung beschreibt, ist deshalb erst auf einem echten Server
# gegangen worden — und dort an einem Werkzeug gescheitert, das im Image gar nicht lag.
set -eu

cd "$(dirname "$0")/.."

project=installationspruefung
tag=pruefung
# Der Port, unter dem die Oberfläche in dieser Prüfung erscheint. Nicht der
# voreingestellte: Auf dem Rechner, der das hier laufen lässt, kann er belegt sein.
port=18081
compose="docker compose -p $project -f docker-compose.yml -f docker-compose.prod.yml"

export IMAGE_TAG=$tag
export POSTGRES_PASSWORD=pruefung
export API_KEY=pruefung
export PUBLIC_URL=http://localhost
export WEB_PORT=$port
export WEB_BIND=127.0.0.1

cleanup() {
    $compose down --volumes --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

cleanup

echo "== Images aus diesem Stand bauen"
# Gebaut wird aus der Entwicklungsfassung, gestartet wird die Betriebsfassung: Geprüft
# werden soll die Datei, die auf dem Server läuft, mit dem Code von hier.
docker build --target api -t "ghcr.io/praetorianer777/behoerdenbarriere-api:$tag" ./backend >/dev/null
docker build -t "ghcr.io/praetorianer777/behoerdenbarriere-frontend:$tag" ./frontend >/dev/null

echo "== Betriebsfassung starten"
# Ohne worker und lighthouse: Die prüfen fremde Websites und brauchen dafür einen
# Browser. Hier geht es um die Installation, nicht um einen Scan.
$compose up -d postgres api frontend

echo "== Behördenliste einspielen"
seeded=$($compose run --rm --entrypoint /seed api)
echo "$seeded"
case "$seeded" in
    *"authorities loaded"*) ;;
    *) echo "Das Einspielen hat nichts gemeldet." >&2; exit 1 ;;
esac

echo "== Die Oberfläche über ihren veröffentlichten Port fragen"
# Über den Port auf dem Host, also genau so, wie der Proxy von seiner Maschine aus
# fragen wird. Und einmal /api hinterher: Dieser Weg — nginx reicht an die API weiter —
# ist schon einmal gebrochen, ohne dass es jemandem aufgefallen wäre.
ask() {
    curl --silent --show-error --fail --max-time 10 --retry 12 --retry-delay 2 \
        --retry-all-errors "http://127.0.0.1:$port$1"
}

ask / >/dev/null
echo "  Die Seite antwortet."

agencies=$(ask "/api/v1/agencies?per_page=1")
total=$(printf '%s' "$agencies" | grep -o '"total":[0-9]*' | head -1 | cut -d: -f2)
if [ "${total:-0}" -lt 100 ]; then
    echo "Die API liefert durch das nginx der Oberfläche $total Behörden: $agencies" >&2
    exit 1
fi
echo "  Die API antwortet durch die Oberfläche, mit $total Behörden."

echo "== Die übrigen Werkzeuge starten wenigstens"
# /import holt seine Daten von Wikidata, /dns aus dem DNS. Beides darf in der CI an
# einem langsamen Dienst scheitern — was hier zählt, ist, dass die Datei da ist und
# anläuft. Genau das war der Fehler, den niemand bemerkt hat, und er sieht anders aus
# als ein Zeitüberschreiten: Docker kommt gar nicht erst zum Start.
starts() {
    tool=$1
    shift
    output=$($compose run --rm --entrypoint "$tool" api "$@" 2>&1 || true)
    case "$output" in
        *"no such file or directory"* | *"executable file not found"*)
            echo "$tool liegt nicht im Image: $output" >&2
            return 1
            ;;
    esac
    echo "  $tool läuft an."
}

starts /import -dry-run -timeout 60s
starts /dns -domain bund.de

echo
echo "Die Installation steht."
