#!/usr/bin/env bash
# PreToolUse on Edit|Write|NotebookEdit and on Bash (git commit): work in this
# repository happens only on a branch that names a GitLab issue, <type>/<n>-<slug>.
# The global bash-guard keeps commits off main; this one goes further and refuses a
# file edit or a commit anywhere that no issue stands behind, so that every change is
# traceable to an issue on gitlab.the-machine.eu.
set -uo pipefail

input="$(cat)"
tool="$(printf '%s' "$input" | jq -r '.tool_name // empty')"
cwd="$(printf '%s' "$input" | jq -r '.cwd // empty')"
root="${CLAUDE_PROJECT_DIR:-${cwd:-.}}"

deny() {
  jq -n --arg r "$1" '{
    hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: $r
    }
  }'
  exit 0
}

issue_branch_re='^[a-z]+/[0-9]+-[a-z0-9-]+$'

hint() {
  printf '%s' "Auf '$1' wird nicht gearbeitet: jede Änderung braucht ein GitLab-Issue und einen Branch, der es nennt.

  glab issue create --title \"<Betreff>\" --description \"<Text>\"
  git switch -c fix/<nr>-<thema>   # oder feature/, chore/, docs/

Bestehende Issues: glab issue list"
}

# The branch is asked where the change happens, not where the project directory stands.
# A worktree under .claude/worktrees/ belongs to the project by its path but carries a
# branch of its own; asked in the main checkout it answered "main", and every change of a
# worktree agent was refused although it sat on a correct issue branch.
check_branch() {
  local dir="$1" branch
  # A file that does not exist yet has no directory of its own; then the nearest one that
  # does exist decides, and it lies in the same worktree.
  while [[ -n "$dir" && ! -d "$dir" ]]; do dir="$(dirname "$dir")"; done
  [[ -d "$dir" ]] || return 0
  git -C "$dir" rev-parse --verify HEAD >/dev/null 2>&1 || return 0
  branch="$(git -C "$dir" rev-parse --abbrev-ref HEAD 2>/dev/null)"
  [[ "$branch" =~ $issue_branch_re ]] || deny "$(hint "$branch")"
}

case "$tool" in
  Edit|Write|NotebookEdit)
    file="$(printf '%s' "$input" | jq -r '.tool_input.file_path // .tool_input.notebook_path // empty')"
    [[ -n "$file" ]] || exit 0
    # Only files of this repository count; the scratchpad and the memory lie outside it.
    case "$file" in
      "$root"/*) ;;
      *) exit 0 ;;
    esac
    check_branch "$(dirname "$file")"
    ;;
  Bash)
    cmd="$(printf '%s' "$input" | jq -r '.tool_input.command // empty')"
    [[ -n "$cmd" ]] || exit 0
    while IFS= read -r part; do
      read -r first _ <<< "$part"
      # The base name decides, not the spelling: /usr/bin/git is git. Written out in full
      # the command used to walk past this check.
      [[ "$(basename -- "$first")" == "git" ]] || continue
      [[ "$part" =~ (^|[[:space:]])commit([[:space:]]|$) ]] || continue
      dir="${cwd:-$root}"
      [[ "$part" =~ -C[[:space:]]+([^[:space:]]+) ]] && dir="${BASH_REMATCH[1]}"
      check_branch "$dir"
    done < <(printf '%s\n' "$cmd" | tr ';|&' '\n')
    ;;
esac

exit 0
