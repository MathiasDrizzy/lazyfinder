# Research 02: file watching on macOS and git consistency

Consulted 2026-10-13. Measurements were run on this Mac (macOS, Go 1.27.2, git 2.56.0) the same day.

## Sources

- fsnotify v1.10.1 (published 2026-05-04), README: backends are inotify, kqueue (BSD/macOS), ReadDirectoryChangesW,
  FEN. FSEvents: "Needs support in x/sys/unix". Recursive watching: "No, you must add watches for any directory you
  want to watch (a recursive watcher is on the roadmap: #18)". kqueue "requires opening a file descriptor for every
  file that's being watched". Editors that replace files atomically drop file-level watches: watch the parent
  directory. <https://github.com/fsnotify/fsnotify>
- lf `watch` uses fsnotify and falls back to periodic reload (`period`). <https://github.com/gokcehan/lf/blob/master/doc.md>
- Watchman: `settle` default 20 ms, `ignore_vcs` default `.git .hg .svn`. <https://facebook.github.io/watchman/docs/config>
- watchexec loads `.gitignore`/`.ignore` and coalesces events. <https://github.com/watchexec/watchexec>
- `git status --porcelain=v2 --branch -z` is the stable machine format; scripts that run it in the background should
  use `git --no-optional-locks status` so they do not take the index lock. <https://git-scm.com/docs/git-status>
- Git's built-in `core.fsmonitor` daemon exists for macOS and Windows. <https://git-scm.com/docs/git-config>
- NOT DOCUMENTED in anything read: which file under `.git/` changes on commit or checkout. Do not depend on watching
  `HEAD` or `index` alone.

## Measurements (2026-10-13)

| Measurement | Result | Command |
|---|---|---|
| fds opened by fsnotify (kqueue) when adding one directory with 2000 files | 2001 (8 before, 2009 after) | scratch Go program, `fsnotify.Add(dir)` then count `/dev/fd` |
| Soft fd limit in this shell | 1 048 576 | `ulimit -n` |
| `kern.maxfilesperproc` / `kern.maxfiles` | 92 160 / 184 320 | `sysctl` |
| `launchctl limit maxfiles` soft limit | 256 | `launchctl limit maxfiles` |
| `git status --porcelain=v2 --branch -z` on a 318-file repo | 10 ms | `time` |
| same on a 2092-file repo | 29 ms | `time` |
| List and `stat` 10 000 entries in Go | 27 ms | scratch Go program, `os.ReadDir` + `Info()` |

## Conclusions

1. **kqueue costs one fd per entry of a watched directory**, not one per directory. Watching a tree recursively with
   fsnotify on macOS spends fds proportional to the number of files, and the soft limit depends on how the process was
   launched (256 from launchd, huge from an interactive shell). Therefore the watcher must watch only the directories
   currently visible or expanded, never the whole tree, and must degrade to polling when `Add` fails.
2. **Git is fast enough to be the source of truth** for status: tens of milliseconds on repos of thousands of files.
   Debounced `git --no-optional-locks status --porcelain=v2 --branch -z` after a filesystem event is more robust than
   interpreting `.git/` internals.
3. **Listing is not a bottleneck:** 10 000 entries cost 27 ms, so a full re-read of a visible directory on each
   debounced event is acceptable and simpler than incremental patching.
4. Design in ADR 0003; implementation is milestone 2.
