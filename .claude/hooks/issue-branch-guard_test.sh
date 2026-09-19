#!/bin/sh
# Plays the branch guard through in a throwaway repository with a worktree: the main
# checkout stands on main, the worktree on an issue branch. The guard has to judge the
# change where it happens, not where the project directory happens to stand — and it has
# to recognise a commit even when it is called by its absolute path.
set -eu
cd "$(dirname "$0")/../.."

hook="$PWD/.claude/hooks/issue-branch-guard.sh"

# Without jq the guard reads no field out of its input, falls through every case and
# permits everything. The allowed cases below would then pass for the wrong reason, so
# the missing tool has to be an error and not a quiet skip.
command -v jq >/dev/null 2>&1 || {
  echo "FAILED: jq fehlt — ohne jq lässt der Wächter alles durch, und dieser Test bewiese nichts" >&2
  exit 1
}
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

main="$work/repo"
git init -q -b main "$main"
git -C "$main" config user.email test@example.org
git -C "$main" config user.name Test
: > "$main/datei.txt"
git -C "$main" add datei.txt
git -C "$main" -c commit.gpgsign=false commit -qm "erster Stand"

# The worktree lies inside the project directory, exactly as the agents' ones do under
# .claude/worktrees/. That is what makes the case sharp: by the path the file belongs to
# the project, by the branch it belongs to the worktree.
side="$main/.claude/worktrees/agent"
git -C "$main" worktree add -q -b fix/1-thema "$side" >/dev/null 2>&1

# The guard answers a denial as JSON with permissionDecision "deny" and says nothing at
# all when it lets something pass.
# The guard is a bash script and is called as one — where /bin/sh is dash or ash, its
# [[ ]] and =~ would not survive.
ask() {
  CLAUDE_PROJECT_DIR="$main" bash "$hook" 2>/dev/null | grep -c '"deny"' || true
}

edit() {
  printf '{"tool_name":"Edit","cwd":"%s","tool_input":{"file_path":"%s"}}' "$1" "$2" | ask
}

bash_command() {
  printf '{"tool_name":"Bash","cwd":"%s","tool_input":{"command":"%s"}}' "$1" "$2" | ask
}

check() {
  want="$1"
  got="$2"
  what="$3"
  [ "$want" = "$got" ] || { echo "FAILED: $what — erwartet $want, bekommen $got" >&2; exit 1; }
}

# The worktree stands on fix/1-thema: everything there is allowed, although the main
# checkout stands on main at the same moment.
check 0 "$(edit "$side" "$side/datei.txt")" "Edit im Arbeitsbaum auf einem Issue-Branch"
check 0 "$(bash_command "$side" "git commit -m x")" "Commit im Arbeitsbaum"
check 0 "$(bash_command "$main" "git -C $side commit -m x")" "Commit mit -C in den Arbeitsbaum"

# The main checkout stands on main: nothing is allowed there.
check 1 "$(edit "$main" "$main/datei.txt")" "Edit im Hauptbaum auf main"
check 1 "$(bash_command "$main" "git commit -m x")" "Commit im Hauptbaum auf main"

# The absolute path is the same commit. Whoever writes it out reaches the same guard.
check 1 "$(bash_command "$main" "/usr/bin/git commit -m x")" "Commit als /usr/bin/git im Hauptbaum"
check 0 "$(bash_command "$side" "/usr/bin/git commit -m x")" "Commit als /usr/bin/git im Arbeitsbaum"

# Outside the repository the guard has nothing to say — the scratchpad and the memory
# lie there.
check 0 "$(edit "$main" "$work/draussen.txt")" "Edit außerhalb des Repositoriums"

echo "issue-branch-guard: alle Fälle wie erwartet"
