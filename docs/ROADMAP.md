# Roadmap

Status: proposal for cerebro's approval, 2026-10-13. Priority is by value to Mathias's stated use (browse, preview,
open, follow a project) and by risk. Each milestone is small and shippable; the look of each is approved through
cerebro with real Ghostty captures.

| Milestone | Theme | Content | Why this order |
|---|---|---|---|
| **M1** | Browse, preview, open | R-01 to R-26 in `SPEC.md`: keyboard and mouse navigation, history, breadcrumb, five sortable columns, sidebar, open-with from config, Space popup (text, image, folder), read-only | The Finder replacement for reading; no risk to data |
| **M2** | Live follow and git | R-30, R-31: watcher (ADR 0003), debounce, polling fallback, git status column and branch, hide ignored, "new" highlight | Mathias's second headline feature; builds on M1's `Listing` |
| **M3** | Tree, search and project mode | R-32, R-33, R-34, R-50, R-52, R-54: `--follow <dir>` narrow tree mode, disclosure triangles, search bar with name, kind, date and size filters (folder and recursive), go to folder, default openers table | Needs the watcher; turns lazyfinder into a VS Code sidebar companion |
| **M4** | File operations | R-35, R-51: copy, move, rename, new folder, move to Trash with undo, selection of many, confirmation rules | Write operations come after the read paths are trusted; every operation gets a dry-run and fixture tests |
| **M5** | Previews++ (done, 0.5.0; Sixel and iTerm2 deferred, ADR 0011) | Sixel and iTerm2 protocols, PDF first page, video/audio metadata, archive listing, Markdown rendering, hex view | Value but each is optional and terminal-dependent |
| **M6** | Productivity | Tabs, bookmarks UI, column view (Miller) as an alternate layout, tags, quick filters, `fd`/`rg` integration, custom commands in config | Convenience features; layout alternates once the core is stable |
| **M7** | Release polish | Homebrew tap, signed releases, completions, man page, i18n of UI strings | Distribution |

Rules for all milestones: test first (rojo, then verde), one ADR per design decision, ED-9 versioning (`0.y.z`), CHANGELOG
entry per user-visible change.

## Known risks

* Bubble Tea v2 has no explicit "stable" statement found (research 03): pin versions; ADR 0001.
* Multiplexers and image protocols: probe with timeout; user can force `off`.
* fd exhaustion with kqueue: bounded by design (ADR 0003).
