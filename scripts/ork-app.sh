#!/bin/sh
# Lance la micro-app ork. Au premier usage d'une version, télécharge le binaire
# depuis les releases GitHub, vérifie sa somme de contrôle et le garde en cache.
#
# ORK_APP_BIN       binaire à utiliser à la place (développement)
# ORK_APP_BASE_URL  source de téléchargement à la place de GitHub (tests)
set -eu

fail() {
  printf 'ork-app : %s\n' "$1" >&2
  exit 1
}

if [ -n "${ORK_APP_BIN:-}" ]; then
  exec "$ORK_APP_BIN" "$@"
fi

script=$0
while [ -L "$script" ]; do
  target=$(readlink "$script")
  case $target in
    /*) script=$target ;;
    *) script=$(dirname "$script")/$target ;;
  esac
done
plugin=$(cd "$(dirname "$script")/.." && pwd)

version=$(sed -n 's/^[[:space:]]*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$plugin/.claude-plugin/plugin.json" | head -n 1)
[ -n "$version" ] || fail "version introuvable dans $plugin/.claude-plugin/plugin.json"

case $(uname -s) in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) fail "système non pris en charge : $(uname -s)" ;;
esac
case $(uname -m) in
  arm64 | aarch64) arch=arm64 ;;
  x86_64 | amd64) arch=amd64 ;;
  *) fail "processeur non pris en charge : $(uname -m)" ;;
esac

# Hors du plugin : il peut être un lien symbolique vers un clone git.
cache=${XDG_CACHE_HOME:-$HOME/.cache}/ork/$version
binary=$cache/ork-app

if [ ! -x "$binary" ]; then
  command -v curl >/dev/null 2>&1 || fail "curl est requis pour télécharger le binaire"
  if command -v sha256sum >/dev/null 2>&1; then
    hash_file() { sha256sum "$1" | cut -d ' ' -f 1; }
  elif command -v shasum >/dev/null 2>&1; then
    hash_file() { shasum -a 256 "$1" | cut -d ' ' -f 1; }
  else
    fail "sha256sum ou shasum est requis pour vérifier le binaire"
  fi

  base=${ORK_APP_BASE_URL:-https://github.com/peufo/ork/releases/download/v$version}
  asset=ork-app-$os-$arch

  mkdir -p "$cache"
  temp=$(mktemp -d "$cache/download.XXXXXX")
  trap 'rm -rf "$temp"' EXIT

  printf 'ork-app : téléchargement de la version %s (%s-%s)\n' "$version" "$os" "$arch" >&2
  curl -fsSL "$base/$asset" -o "$temp/$asset" \
    || fail "binaire introuvable : $base/$asset. La release v$version est-elle publiée ?"
  curl -fsSL "$base/checksums.txt" -o "$temp/checksums.txt" \
    || fail "sommes de contrôle introuvables : $base/checksums.txt"

  expected=$(awk -v name="$asset" '$2 == name { print $1 }' "$temp/checksums.txt")
  [ -n "$expected" ] || fail "$asset absent de checksums.txt"
  actual=$(hash_file "$temp/$asset")
  [ "$expected" = "$actual" ] || fail "somme de contrôle incorrecte pour $asset : rien n'est installé"

  chmod 755 "$temp/$asset"
  mv "$temp/$asset" "$binary"
  rm -rf "$temp"
  trap - EXIT
fi

exec "$binary" "$@"
