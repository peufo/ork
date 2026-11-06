#!/bin/sh
# Builds the micro-app from source and opens it on a project. Ctrl-C stops it.
# Usage: sh app/run.sh [project path, current directory by default]
set -eu

repo=$(cd "${1:-.}" && pwd)
cd "$(dirname "$0")"
bin="${TMPDIR:-/tmp}/ork-app-dev"
go build -o "$bin" .
ORK_APP_BIN="$bin" exec sh ../scripts/ork-app.sh --repo "$repo" --open
