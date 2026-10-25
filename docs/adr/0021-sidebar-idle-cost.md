# What the sidebar does while it is idle

* Status: accepted
* Date: 2026-10-24

## Context and problem statement

Eight herdr sidebars (one per tab) sat at 7-11% CPU each while idle. Measured with a pty and an instrumented copy of the
program: 130 watcher events in 12 s, all of them `CHMOD .git/index`. On macOS, reading a file updates its access time and
kqueue reports that as an attribute change; `git status` reads the index, so every scan woke the watcher, which reloaded
the folder and ran `git status` again, without end. A sidebar in a folder that is not a repository, with the same herdr
poll, used 0.1%.

## Decision

* The watcher ignores attribute-only events of the files in a `.git` folder (`internal/watch/fs.go`, `readOnlyTouch`).
  What git really changes arrives as a write, a create or a rename.
* A change of the `.git` folder reloads only when its files (name, size, modification time) differ from the last scan
  (`gitDirSignature`).
* Whatever the cause, git is read at most once per `Deps.GitMinGap` (one second) when `.git` changes; the changes that
  arrive in between are read once, when the gap ends.
* The herdr poll asks every 500 ms while the folder of the other pane changes or the user is in the sidebar, and every
  second once 20 answers in a row did not change (`herdrDelay`). It is one call over a unix socket; measured, it costs
  nothing in lazyfinder and about 1% of one core in the herdr server for eight sidebars, so the backoff only halves
  that. A slower fixed poll was rejected: it would delay the sidebar following a `cd` all the time.
* Not done, on purpose: caching the settings that `readOnly` reads (`internal/git/trust.go`) between reads. A read is three
  git processes (`rev-parse`, `config`, `status`) and, with the changes above, happens only when something changed. A
  cache keyed on `.git/config` would miss an include that did not exist yet, a worktree config and a changed global
  config, and that check is what keeps a hostile repository from running its own filters (ADR 0019); one process less
  is not worth that risk.

## Consequences

* An idle sidebar in a repository stays at about 0% CPU, and several sidebars in one repository no longer wake each other.
* A change of git state shows up at most one second late when another change came just before it.
* Sidebars started before the update keep the old binary until their pane is restarted.
