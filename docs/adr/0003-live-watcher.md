# Live watcher design

* Status: accepted (2026-10-13, approved by cerebro with the phase 1 audit) (implemented in milestone 2; M1 ships only the interface)
* Date: 2026-10-13

## Context and problem statement

Mathias wants to follow a project while an agent creates files, "que sea consistente con git y salgan las cosas
nuevas", without manual refresh. Research 02 measured that fsnotify's kqueue backend on macOS opens one file
descriptor per entry of each watched directory (2001 fds for one directory of 2000 files) and has no recursion.

## Decision drivers

* Never exhaust file descriptors on a large tree.
* Consistent with git: status and ignore rules come from git itself.
* Works when watching fails (fd limit, network volume, FUSE).

## Considered options

1. fsnotify recursively over the whole tree.
2. fsnotify on only the directories currently displayed or expanded, plus polling fallback.
3. FSEvents through cgo or a third-party binding.
4. Polling only.

## Decision outcome

Chosen option: **2**.

* `watch.Watcher` interface: `Watch(dir) / Unwatch(dir)` and an events channel carrying `Changed(dir)` only. The
  view reacts by re-reading that directory (27 ms for 10 000 entries, so no incremental patching).
* Watches exist only for directories currently visible (current dir, expanded dirs in follow mode, capped at a
  configurable `watch.max_dirs`, default 64). Leaving a directory unwatches it.
* **Debounce** 50 ms trailing, coalescing bursts (an agent writing many files).
* **Fallback:** if `Add` returns an error (EMFILE, unsupported), or the dir is on FUSE/network, switch that dir to
  polling every `watch.poll_interval` (default 2 s) comparing `ReadDir` results; surface "polling" in the status line.
* Ignore noise: `.DS_Store`, editor swap files, and Spotlight churn are filtered before the debounce.
* **Git:** after a debounced event, run `git --no-optional-locks status --porcelain=v2 --branch -z` in the repo root
  (found once per directory with `git rev-parse --show-toplevel`) and merge the result into the listing as `GitStatus`.
  Measured 10 ms (318 files) and 29 ms (2092 files). Never parse `.git/` internals (no documented file changes on
  commit/checkout); a slow poll (default 5 s) of git status in follow mode catches commits and checkouts made
  elsewhere. `.gitignore` is respected by asking git, not by reimplementing it.
* Rejected: option 1 (fd cost proportional to files), 3 (cgo, no maintained Go binding verified), 4 (latency and
  idle cost).

### Consequences

* Good: bounded fd use; git is the single source of truth; graceful degradation.
* Bad: a new file deep inside a collapsed folder does not appear until the folder is opened or the git poll runs;
  documented behavior, indicated by git status counts in the parent row (M2).
* To measure in M2: end-to-end latency from file creation to screen (target under 1 s, R-30) and fd count under load.
