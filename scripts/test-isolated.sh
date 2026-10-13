#!/usr/bin/env bash
# Runs the Go tests with HOME and the XDG folders pointing at an empty temporary directory (SPEC R-23): a test that
# reads the real home, the real config or real files fails here. The Go caches stay where they were.
set -euo pipefail
cd "$(dirname "$0")/.."

cache="$(go env GOCACHE)"
modcache="$(go env GOMODCACHE)"
tmp="$(mktemp -d "${TMPDIR:-/tmp}/lazyfinder-home.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT

env HOME="$tmp" XDG_CONFIG_HOME="$tmp/.config" XDG_DATA_HOME="$tmp/.local/share" XDG_STATE_HOME="$tmp/.local/state" \
  XDG_CACHE_HOME="$tmp/.cache" LAZYFINDER_CONFIG="" GOCACHE="$cache" GOMODCACHE="$modcache" \
  go test -count=1 ./...
