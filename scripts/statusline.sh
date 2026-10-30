#!/bin/sh
# Status line Claude Code : ce qui attend le développeur dans les tickets ork.
# N'affiche rien hors d'un dépôt qui contient doc/ork.md.

dir=$(jq -r '.workspace.current_dir // .cwd // empty' 2>/dev/null)
[ -n "$dir" ] || exit 0

# Dépôt principal, même depuis un worktree d'ouvrier : les tickets n'y vivent que là.
common=$(git -C "$dir" rev-parse --path-format=absolute --git-common-dir 2>/dev/null) || exit 0
repo=$(dirname "$common")
[ -f "$repo/doc/ork.md" ] || exit 0

blocked="" plan="" review=0 running=0
for f in "$repo"/doc/tickets/*.md; do
  [ -f "$f" ] || continue
  set -- $(awk '
    NR == 1 && $0 != "---" { exit }
    NR > 1 && $0 == "---" { exit }
    /^id:/ { id = $2 }
    /^status:/ { st = $2 }
    END { print id, st }
  ' "$f")
  id=$1 st=$2
  case $st in
    blocked) blocked="$blocked $id" ;;
    plan-review) plan="$plan $id" ;;
    in-review) review=$((review + 1)) ;;
    planning|in-progress) running=$((running + 1)) ;;
  esac
done

out="ork"
[ -n "$blocked" ] && out="$out · ⏸ bloqué :$blocked"
[ -n "$plan" ] && out="$out · 📋 plan :$plan"
[ "$review" -gt 0 ] && out="$out · 👀 $review à relire"
[ "$running" -gt 0 ] && out="$out · ⚙ $running en cours"
[ "$out" = "ork" ] && out="ork · rien ne t'attend"
printf '%s\n' "$out"
