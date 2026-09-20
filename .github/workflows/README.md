# Workflows

`ci.yml` läuft bei jedem Push auf `main` und bei jedem Pull Request:

- **backend** — Formatierung, `go vet`, alle Tests mit Race-Detector. Postgres und ein
  headless Chrome laufen als Service-Container, damit die Datenbank- und Browsertests
  nicht übersprungen werden; genau die prüfen, was Unit-Tests nicht können.
- **frontend** — Lint, Vitest (inklusive axe über das gerenderte DOM), Build.
- **accessibility** — die gebaute Seite wird ausgeliefert und mit unserem eigenen
  Scanner geprüft. Ein Verstoß lässt den Lauf scheitern.

Die Cross-Check-Prüfung gegen `@axe-core/cli` (`AXE_CLI_CROSSCHECK=1`) und der
Erreichbarkeitstest der Seed-Liste (`SEEDS_NETWORK_TEST=1`) laufen nicht in der CI: sie
greifen auf fremde Server zu. Sie gehören vor eine Änderung an Scanner oder Seed-Liste.
