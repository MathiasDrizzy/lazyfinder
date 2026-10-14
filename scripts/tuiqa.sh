#!/usr/bin/env bash
# Runs every scenario in testdata/tuiqa against a freshly built binary, in a pty with an isolated HOME.
# The runner is the tuiqa pty harness: set TUIQA_DIR to the folder of its Go module (with cmd/tuiqa).
# It is not part of this repository, so CI does not run this script; verificar.sh does, on the maintainer's machine.
# Without the harness the script fails, unless TUIQA_OPTIONAL=1.
set -euo pipefail
cd "$(dirname "$0")/.."

TUIQA_DIR="${TUIQA_DIR:-}"
if [ -z "$TUIQA_DIR" ] || [ ! -d "$TUIQA_DIR" ]; then
  echo "tuiqa: set TUIQA_DIR to the folder of the tuiqa harness (now: '${TUIQA_DIR}')" >&2
  [ "${TUIQA_OPTIONAL:-0}" = 1 ] && exit 0
  exit 1
fi

out="$(mktemp -d "${TMPDIR:-/tmp}/lazyfinder-tuiqa.XXXXXX")"
trap 'rm -rf "$out"' EXIT
go build -o "$out/lazyfinder" ./cmd/lazyfinder
(cd "$TUIQA_DIR" && go build -o "$out/tuiqa" ./cmd/tuiqa)

fails=0
for f in testdata/tuiqa/*.json; do
  if LAZYFINDER_BIN="$out/lazyfinder" "$out/tuiqa" run "$f" -out "$out/res" >"$out/log" 2>&1; then
    echo "ok    $(basename "$f")"
  else
    echo "FAIL  $(basename "$f")"; tail -5 "$out/log"; fails=$((fails + 1))
  fi
done
exit "$fails"
