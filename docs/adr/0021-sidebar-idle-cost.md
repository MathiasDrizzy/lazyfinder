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
* The same limit holds for a change of the folder itself (2026-10-26). A burst of changes (an agent writing twenty files a
  second) used to run a `git status` for each reload, about 56 git processes a second, 10% of the sidebar and more of
  git. The rows still follow the folder at once; the marks are read at most once per `Deps.GitMinGap`, and once more
  when it ends (`gitLateMsg`). Measured with a pty and a `git` that counts calls: 56.1 to 3.0 processes a second at
  twenty files a second, 11.6 to 3.0 at four edits of a tracked file, and a lone new file still shows, marked, in 0.1 s.
  A refresh the user asks for and entering a folder read git at once.
* The sidebar draws at most 15 times a second (`programFPS`, 2026-10-27). Bubble Tea runs its renderer on a ticker of 60
  beats a second for as long as the program lives, so an idle sidebar woke 60 times a second whatever it showed. A key
  shows within 50 ms instead of 15 (measured with a pty: median 46 ms against 13). Idle CPU of a sidebar, six at a time,
  with the plugin environment: 0.70% to 0.31%; without herdr: 0.38% to 0.13%. The window of the normal mode keeps 60.
* The poll of the folder of the other pane is one `pane.list` (`Client.List`). Since 0.16.10 it also asked
  `pane.process_info` for every pane that runs no agent, to tell the sidebars; only the hooks need that (`Client.Panes`).
* Not done, on purpose: following the other pane by herdr events instead of polling. herdr emits focus, title and
  layout events but none for a change of folder (`cd`), so a slower safety poll would remain, and a long-lived socket
  reader with its `events_lost` recovery would add more than the roughly 0.15% it would save.
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
* A change of git state shows up at most one second late when another change came just before it; a new file in a burst
  has no mark for up to a second.
* Sidebars started before the update keep the old binary until their pane is restarted.
