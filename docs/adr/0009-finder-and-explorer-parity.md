# Parity with Finder and with the Windows Explorer search

* Status: accepted (2026-10-13, order 2026-10-13-03 relayed by cerebro: "debe tener las mismas funcionalidades que
  finder y el buscador de archivos de windows, tiene que tener protocolo kitty y se debe poder hacer click y
  configurar con que programas abrir los archivos, deben venir unos por defectos obviamente, debe ser todo
  configurable")
* Date: 2026-10-13

## Context and problem statement

Milestones 1 and 2 cover browsing, preview, open-with, theming, live follow and git. Mathias now asks for feature
parity with Finder and with the file search of Windows Explorer, with the Kitty protocol, clicks, default openers
that can all be replaced, and everything configurable. The ROADMAP needs a feature matrix so each item has a
milestone and a test, instead of a vague "same as Finder".

## Decision drivers

* Each feature is small enough to test with a unit test or a tuiqa scenario.
* The terminal is not a GUI: some Finder features (icon view, Cover Flow, drag to the desktop) have no terminal
  equivalent and are listed as out of scope instead of being faked.
* Writes come after reads are trusted (ADR 0002 and the ROADMAP order).

## Decision outcome

Adopt the matrix below. "Done" means shipped, otherwise the milestone that owns it. The requirement numbers are in
`SPEC.md` (R-50 and later).

| Area | Finder | Explorer search | lazyfinder | Milestone |
|---|---|---|---|---|
| Browse | columns list, sidebar, path bar, back/forward, Go to Folder | address bar | done (R-01..R-26); `Go to folder` prompt | M1, M3 |
| Sort and columns | by any column | same | done | M1 |
| Preview | Quick Look | preview pane | Space popup, Kitty images, Quick Look fallback | M1, M5 |
| Open with | default app, Open With | same | configurable per extension, defaults shipped | M1 |
| Live view | auto refresh | auto refresh | done (watcher) | M2 |
| Tree | disclosure triangles | tree pane | expand in place (R-33) | M3 |
| Search | Spotlight scoped search | search box with filters | `/` bar: name, kind, date, size filters, live results, this folder or recursive (R-34, R-50) | M3 |
| Selection | click, shift, cmd | same | many with `v`/ctrl-click, select all (ADR 0010) | M4 |
| Operations | copy, move, rename, new folder, Trash, duplicate, Get Info | same | same, Trash only, undo (R-35, R-51) | M4 |
| Clipboard | copy/paste files | same | copy, cut and paste inside the app | M4 |
| Previews++ | Quick Look of PDF, media, archives | preview pane | Sixel/iTerm2, PDF, archives, Markdown, hex | M5 |
| Tabs and views | tabs, column view | tabs | tabs, Miller columns | M6 |
| Tags and smart lists | tags, smart folders | saved searches | tags in extended attributes, saved searches | M6 |
| Recents | Recents | Recent files | recents from the history | M6 |
| Distribution | | | Homebrew tap, completions, man page | M7 |

Search design:

* The query is one line: free words match the name (case-insensitive substring, or a glob with `*`), and filters are
  `key:value` tokens: `kind:image|text|folder|<extension>`, `modified:today|week|month|year|>YYYY-MM-DD|<YYYY-MM-DD`,
  `size:>10MB|<1KB`, `name:`/`ext:`. Same filters are offered as clickable chips so the mouse reaches everything.
* Scope toggles between the current folder and recursive; the recursive walk runs in a goroutine, streams results
  in batches, can be cancelled with `esc`, skips what is ignored by git when `git.hide_ignored` is on, and never
  follows symlinks into a loop.
* Results are a normal list with an extra "Where" column, so open, preview, sort, open-with and clicks all work on
  them.

Out of scope (terminal limits, not oversights): icon and gallery views, drag and drop to other apps, AirDrop,
iCloud, Cover Flow. Rejected: calling Spotlight (`mdfind`) as the engine; it is macOS-only and ignores the filters we
need, so a Go walker is used and `mdfind` may be added later as an optional accelerator.

### Consequences

* Good: a testable checklist; one result view for every search.
* Bad: a recursive walker on a huge tree costs time; mitigated by streaming, cancel and a result cap
  (`search.max_results`, default 5000).
