#!/usr/bin/env bash
# Rewrites the postflight block GoReleaser v2.18 writes in the tap's cask as postflight_steps (tools/caskfix), so
# Homebrew stops warning that postflight is deprecated (order 2026-10-22-17). It changes the tap only when the cask
# still has that block. Needs GH_TOKEN with write access to the tap. Remove it with tools/caskfix once GoReleaser
# writes postflight_steps itself (goreleaser/goreleaser#6873, planned for v2.19).
set -euo pipefail
cd "$(dirname "$0")/.."

api="repos/MathiasDrizzy/homebrew-tap/contents/Casks/lazyfinder.rb"
work="$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/caskfix.XXXXXX")"
trap 'rm -rf "${work:?}"' EXIT

gh api "$api?ref=main" >"$work/cask.json"
jq -r .content "$work/cask.json" | base64 -d >"$work/cask.rb"
go run ./tools/caskfix <"$work/cask.rb" >"$work/fixed.rb"
if cmp -s "$work/cask.rb" "$work/fixed.rb"; then
  echo "cask: already uses postflight_steps"
  exit 0
fi
gh api -X PUT "$api" \
  -f message="fix(lazyfinder): use postflight_steps instead of postflight" \
  -f content="$(base64 <"$work/fixed.rb" | tr -d '\n')" \
  -f sha="$(jq -r .sha "$work/cask.json")" \
  -f branch=main >/dev/null
echo "cask: postflight rewritten as postflight_steps"
