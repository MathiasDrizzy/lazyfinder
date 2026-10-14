# CI minutes: Linux on every push, macOS before a release

* Status: accepted (2026-10-14, order 2026-10-14-12)
* Date: 2026-10-14

## Context and problem statement

The repository is private, so GitHub Actions minutes come out of the account's monthly allowance, and a macOS runner
counts ten times its real minutes. Every push ran the tests on Linux and macOS: a run took about 4 minutes on macOS
(40 counted) and 10 on Linux, and the macOS share was most of the month's use. When the allowance runs out, GitHub
stops all CI until the next month.

## Decision

* `ci.yml` runs on Linux only, on every push to `main` and `ci/**` and on pull requests, with
  `cancel-in-progress` so a newer push cancels the run of an older one, a timeout per job, and the Go cache.
* Changes that touch only documentation (`**.md`, `docs/**`, `assets/readme/**`, `LICENSE`) run no CI.
* `ci-macos.yml` runs `gofmt`, `go vet` and `go test -race` on macOS on the push of a `v*` tag (next to
  `release.yml`) and by hand (`workflow_dispatch`). It does not use the Go cache: a tag build reads nothing other runs
  wrote.
* macOS is still tested on every change: the maintainer's local check runs the whole suite on a Mac before each push.

## Consequences

* A change that breaks only on macOS and is pushed without `verificar.sh` is caught at the next tag or manual run,
  not at the push. The push hook requires `verificar.sh` for `main`, so this needs a push that skips the hook.
* A documentation change that breaks a Go test that reads the docs (the QA matrix test reads `docs/`) is caught by
  `verificar.sh` before the push, not by CI.
* Before a release, run `gh workflow run ci-macos.yml` on the commit to tag, so the macOS result is known before the
  tag and not only next to it.
