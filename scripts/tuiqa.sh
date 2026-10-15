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

# The scenarios run the real binary on this machine: Enter on a file would launch the real open, which opens apps,
# Finder or Archive Utility on the desktop. Fake openers go first in PATH and log each call; any call fails the run.
mkdir -p "$out/fakebin"
for p in open qlmanage osascript xdg-open; do
  # shellcheck disable=SC2016 # $0 and $* belong to the fake script, expanded when it runs
  printf '#!/bin/sh\nprintf "%%s %%s\\n" "$(basename "$0")" "$*" >>"%s"\nexit 0\n' "$out/opener.log" >"$out/fakebin/$p"
  chmod +x "$out/fakebin/$p"
done
# The clipboard tools are fakes too: a right click on the breadcrumb copies, and no scenario reaches the real clipboard.
for p in pbcopy wl-copy xclip xsel; do
  # shellcheck disable=SC2016 # $0 belongs to the fake script, expanded when it runs
  printf '#!/bin/sh\n{ printf "%%s " "$(basename "$0")"; cat; echo; } >>"%s"\n' "$out/clipboard.log" >"$out/fakebin/$p"
  chmod +x "$out/fakebin/$p"
done
export PATH="$out/fakebin:$PATH"
# Inside herdr the plugin variables would point at the real server: the scenarios run outside it.
unset HERDR_ENV HERDR_SOCKET_PATH HERDR_PANE_ID HERDR_TAB_ID HERDR_WORKSPACE_ID HERDR_PLUGIN_ID HERDR_SESSION
# tuiqa puts its own fake openers first unless told not to; ours are fakes too, and their log is what this script checks
export TUIQA_REAL_OPENERS=1
# Scenarios that open a file on purpose (the fake opener takes the call); any other call fails the run.
opens_on_purpose=" key-combinations.json "

fails=0
for f in "${TUIQA_SCENARIOS:-testdata/tuiqa}"/*.json; do
  if LAZYFINDER_BIN="$out/lazyfinder" "$out/tuiqa" run "$f" -out "$out/res" >"$out/log" 2>&1; then
    echo "ok    $(basename "$f")"
  else
    echo "FAIL  $(basename "$f")"; tail -5 "$out/log"; fails=$((fails + 1))
  fi
  if [ -s "$out/opener.log" ] && [[ $opens_on_purpose != *" $(basename "$f") "* ]]; then
    echo "FAIL  $(basename "$f") launched an opener (a scenario must not open apps):"; cat "$out/opener.log"
    fails=$((fails + 1))
  fi
  : >"$out/opener.log"
  if [ -s "$out/clipboard.log" ]; then
    echo "      copied: $(cat "$out/clipboard.log")"
    [ -n "${TUIQA_CLIPBOARD_LOG:-}" ] && cat "$out/clipboard.log" >>"$TUIQA_CLIPBOARD_LOG"
  fi
  : >"$out/clipboard.log"
done
exit "$fails"
