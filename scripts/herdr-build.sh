#!/bin/sh
# Build step of herdr-plugin.toml: puts the lazyfinder the plugin runs in bin/lazyfinder. With Go it builds this
# checkout; without it, it links the lazyfinder on PATH when that one knows --herdr.
# POSIX sh has no pipefail in every shell (dash), and grep -q below ends the pipe early on purpose.
set -eu

root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
mkdir -p "$root/bin"

if command -v go >/dev/null 2>&1; then
  cd "$root"
  exec go build -trimpath -o bin/lazyfinder ./cmd/lazyfinder
fi

if installed=$(command -v lazyfinder 2>/dev/null) && "$installed" --help 2>&1 | grep -q -- '-herdr'; then
  ln -sf "$installed" "$root/bin/lazyfinder"
  exit 0
fi

echo "lazyfinder plugin: needs Go (https://go.dev/dl) or lazyfinder 0.14 or later on PATH" >&2
echo "  brew install mathiasdrizzy/tap/lazyfinder" >&2
exit 1
