---
name: gh
description: Work with this project's GitHub repository (praetorianer777/behoerdenbarriere) through the gh CLI - list, read, create and comment on issues, push a branch and open a pull request, watch Actions runs and read job logs. Use when the user mentions issues, PRs, pipelines, CI, job logs or pushing.
---

# GitHub via gh

Das Projekt liegt auf `github.com/praetorianer777/behoerdenbarriere` (Remote `origin`).
Alle Befehle laufen im Repo, `gh` nimmt das Repo aus dem Remote;
`-R praetorianer777/behoerdenbarriere` ist nur von außerhalb nötig.

## Vorab

```
gh auth status
```

Ohne Token: den Nutzer bitten, sich selbst anzumelden (`! gh auth login`). Niemals
einen Token einfügen oder speichern.

## Jede Änderung hat ein Issue

Gearbeitet wird nur auf einem Branch `<type>/<n>-<slug>`, wobei `n` die Issue-Nummer ist
(`fix/3-score-normierung`, `feature/1-grundgeruest`). Der Hook
`.claude/hooks/issue-branch-guard.sh` verweigert Dateiänderungen und Commits auf jedem
anderen Branch, `main` eingeschlossen. Also: Issue suchen oder anlegen, mit dessen Nummer
von `main` abzweigen, dann arbeiten. Im PR-Text auf das Issue verweisen (`Closes #<n>`).

## Lesen (ohne Rückfrage)

```
gh issue list
gh issue view <n> --comments
gh pr list
gh pr view <n> --comments
gh run list --limit 10
gh run view <id> --log-failed
```

## Schreiben (vorher fragen)

```
gh issue create --title "<Betreff>" --body "<Text>"
gh issue comment <n> --body "<Text>"
git push -u origin <branch>
gh pr create --fill --base main
gh pr comment <n> --body "<Text>"
```

Push und PR sind nach außen sichtbar — erst fragen, außer der Nutzer hat es für diesen
Schritt schon freigegeben.
