# Lint baseline

* Status: accepted
* Date: 2026-10-13

## Context and problem statement

`golangci-lint` v2 with the `standard` linters (errcheck, govet, ineffassign, staticcheck, unused) is the quality
gate of the development standard (ED, Go section). Enabling every staticcheck rule rewrites working code for style.

## Decision outcome

`.golangci.yml` is the one lazymark already uses: `default: standard`, staticcheck `all` minus the style-only
checks (QF1001, QF1008, ST1018 and the package comment and naming rules ST1000, ST1003, ST1005, ST1016, ST1020,
ST1021, ST1022), `errcheck` ignoring `fmt.Fprint*` (best-effort terminal output) and ignored in `_test.go`, and
`gofmt` as the formatter. The command is pinned (`golangci-lint@v2.14.0`, `govulncheck@v1.9.0`) and is the same in
`verificar.sh` and in CI (ED-19, ED-21).

### Consequences

* Good: the same rules as lazymark, a known baseline, nothing silently unpinned.
* Bad: the style checks are off; a reviewer has to catch naming and comment style.
