# Research 01: existing TUI file managers

Consulted 2026-10-13. Release dates come from the GitHub releases API (confirmed). Stars, licenses and feature
claims come from README/doc summaries and are approximate unless marked otherwise. `NOT DOCUMENTED` means the
pages read did not say; it does not mean the feature is absent.

| Tool | Language | License | Latest release (API) | Notes |
|---|---|---|---|---|
| [yazi](https://github.com/sxyazi/yazi) | Rust | MIT | v26.9.1, 2026-09-01 | Reference implementation: 3-pane Miller view, `linemode` columns, `[open]` rules by MIME or name, configurable mouse events, image previews (kitty, sixel, iTerm2) |
| [superfile](https://github.com/yorukot/superfile) | Go | MIT | v1.6.0, 2026-06-07 | Closest to a "modern" look; v1.6.0 notes mention "configurable sidebar behavior" and "richer file metadata" |
| [lf](https://github.com/gokcehan/lf) | Go | MIT | r42, 2026-07-31 | `watch` option (fsnotify, off by default), `info` columns, mouse off by default |
| [nnn](https://github.com/jarun/nnn) | C | BSD-2 | v5.3, 2026-08-15 | Minimal; plugin/FIFO preview; no column model |
| [broot](https://github.com/Canop/broot) | Rust | MIT | v1.61.0, 2026-10-03 | Best documented git integration (`:gf`, `:gs`, ignored files) |
| [xplr](https://github.com/sayanarijit/xplr) | Rust | MIT | v1.1.2, 2026-09-15 | Hackable; little on columns or mouse |
| [vifm](https://github.com/vifm/vifm) | C | GPL-2.0+ | v0.14.4, 2026-05-31 | Vim-like; previews through plugins |
| [ranger](https://github.com/ranger/ranger) | Python | GPL-3.0 | no GitHub release (NOT VERIFIED: 1.9.4) | Miller view; previews via `scope.sh` |
| [joshuto](https://github.com/kamiyaa/joshuto) | Rust | LGPL-3.0 | NOT VERIFIED | `mimetype.toml` for openers |
| termscp, fx | Rust, Go | MIT | n/a | Not local file managers (remote transfer client, JSON viewer): excluded |

## Findings relevant to lazyfinder

1. **Nobody documents a "Kind" column, a Favorites/Locations sidebar, or live follow tied to git.** The Finder-like
   features Mathias asked for are gaps in the market, not things to copy. (Sources: yazi docs
   <https://yazi-rs.github.io/docs/configuration/yazi>, lf `doc.md`, superfile.dev, consulted 2026-10-13.)
2. **Columns:** yazi `linemode` and lf `info` are the prior art: a configurable list of fields. Copy the idea as an
   ordered `columns` list in `config.json`; add `kind`, which neither has.
3. **Open with:** yazi `[open]` rules (by MIME or file name, named openers) are the prior art. Copy named openers and
   per-extension rules; skip MIME globbing in milestone 1 (extension is what Mathias asked for).
4. **Mouse:** yazi exposes click, scroll, move and drag; lf has click-to-select and right-click-to-open, off by
   default. Lazyfinder makes the mouse a first-class input, on by default (Mathias: "donde pueda hacer click").
5. **Watching:** lf `watch` (fsnotify, off by default, documented FUSE caveat, `period` reload as fallback) is the only
   documented watcher. Yazi's behavior is not documented.
6. **Git:** broot shows branch, ahead/behind and ignored files; `git.yazi` shows status as a column. Copy "status as a
   column" for milestone 2.

## What we copy, what we do not

Copy: 3-pane/Miller idea as an *option* later; configurable columns; named openers with per-extension rules; git
status as a column; lf's "watch off by default plus polling fallback" as a safety valve.

Do not copy: update checks on by default (superfile `auto_check_update`: network calls are opt-in here); previews that
depend on many external tools (ranger `scope.sh`: previews work with zero external tools, Quick Look is only a
fallback); nnn/xplr as UI references (no column or mouse depth).

Not verified: sorting and mouse details for ranger, joshuto, superfile, xplr, vifm; yazi's watch behavior.
