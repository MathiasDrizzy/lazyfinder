# lazyfinder specification

lazyfinder is a lazy file manager for the terminal: a Finder-like TUI with columns, preview, open-with and live
follow mode, in the spirit of lazymark: stop using Finder and still have everything needed, in a terminal app.

## How to read this document

- **Source requirements** `Q1`–`Q8` summarize the original request (2026-10-13). They are requirements.
- **PROPOSAL** items were deductions from them, kept to show where a requirement came from.
- Every requirement `R-nn` has an acceptance criterion `AC` that a test or a `tuiqa` scenario checks. Rule of work:
  the test is written first and seen failing against the previous commit (ED-22, ED-25).
- Milestone tags: `M1` first usable release, `M2` live follow and git, `M3+` later milestones.
- The reference screenshot (a Finder window) shows: Favorites/Locations sidebar, columns Name, Date Modified,
  Date Created, Size, Kind, folders with a `>` disclosure, a breadcrumb path at the bottom, back/forward buttons and
  a Search field.

## Source requirements

| Id | Requirement |
|---|---|
| Q1 | A lazy file browser, very close to the macOS Finder or the Windows file explorer. |
| Q2 | Click to open files, and choose what opens each type: a `.txt` with TextEdit, VS Code or any editor; an `.xlsx` with Excel or LibreOffice; a picture with Preview. |
| Q3 | Press space for a Quick Look-like preview popup. |
| Q4 | Reach a folder by clicking through the folders above it. |
| Q5 | A live follow mode, like the VS Code sidebar, to watch a project while an agent creates files: new files appear without a refresh, consistent with git. |
| Q6 | A terminal app with the nature of lazymark that replaces Finder for everyday use. |
| Q7 | Room for much more. |
| Q8 | The look of the macOS Finder with the lazy style of lazymark, and everything configurable: clicks, themes, languages, settings, the Kitty graphics protocol. Reuse what lazymark already solved. |

## Milestone 1 requirements

### Navigation (Q1, Q2, Q4)

- **R-01 Keyboard navigation (M1).** Arrow keys and `hjkl` move the selection; `Enter` or `l`/`Right` enters a
  directory or opens a file; `Backspace` or `h`/`Left` goes to the parent. AC: scenario sends each key against a
  fixture tree and the screen shows the expected directory and selection.
- **R-02 Mouse navigation (M1)** (Q2, Q4). Single click selects a row; double click enters a directory or opens a
  file; the wheel scrolls the list; clicks on sidebar items, breadcrumb segments and toolbar buttons act on them.
  AC: pty scenario with SGR mouse sequences reaches a nested fixture folder using clicks only.
- **R-03 History (M1).** Back and forward buttons (clickable) and keys (`Alt+Left`/`Alt+Right`, `[`/`]`) walk a
  per-session history; navigating after going back discards the forward stack. AC: unit test on the history type
  plus scenario.
- **R-04 Breadcrumb (M1).** The current path is shown as clickable segments at the bottom, as in the reference
  screenshot. AC: clicking segment N changes directory to the Nth ancestor; long paths elide the middle, never the
  last segment. A right click on a segment copies the path of that folder to the clipboard (pbcopy, wl-copy, xclip or
  xsel; OSC 52 when none is installed) and the status line says what was copied.

### List and columns (Q1)

- **R-05 Columns (M1).** The list has the columns Name, Date Modified, Date Created, Size, Kind, in the order and
  visibility set by `config.json` (`columns`). Directories show size as `--`. Sizes use SI-style human units like
  Finder. AC: golden of a fixture directory at 120x40; unit tests for size and date formatting.
- **R-06 Sorting (M1).** Sort by any column ascending or descending: click on the header, or keys `s`/`S`
  (cycle column / reverse). Directories-first is on by default and configurable. Sorting by name is natural and
  case-insensitive. AC: unit tests on the comparator; scenario clicks a header twice and checks order.
- **R-07 Kind (M1).** Kind comes from a built-in extension table (e.g. `Folder`, `Plain Text`, `PDF Document`,
  `JPEG image`, `Alias` for symlinks, `Document` fallback), not from spawning processes. AC: table-driven test.
- **R-08 Hidden files (M1, PROPOSAL).** Dotfiles are hidden; `.` toggles them. AC: scenario.

### Sidebar (Q1)

- **R-09 Sidebar (M1).** A left sidebar lists **Favorites** and **Locations** read from `config.json`; defaults:
  Home, Desktop, Documents, Downloads, Applications under Favorites; `/` and mounted volumes (`/Volumes/*`) under
  Locations. On Linux: Home, Desktop, Documents, Downloads under Favorites; `/` (Computer) and the Trash under
  Locations. Click or `Enter` jumps there; `Tab` moves focus between sidebar and list. Missing paths are shown
  dimmed and are not an error. AC: scenario with a fixture config; unit test for missing paths.

### Open with (Q2)

- **R-10 Open with (M1).** `config.json` maps extensions to named openers; a file opens with the opener of its
  extension, otherwise with the system default (`open <file>` on macOS). Examples from Q2: `txt` to a text editor,
  `xlsx` to Excel or LibreOffice, images to Preview. AC: opener resolution unit tests; the integration test uses a
  recording stub as the opener, never a real application.
- **R-11 Open With menu (M1).** `O` (or right click) lists the configured openers for the file and the default;
  choosing one opens the file with it. AC: scenario against the recording stub.
- **R-12 Nothing runs unless configured (M1).** Openers are argv arrays (`["code", "{file}"]`), executed directly
  without a shell; `{file}` is passed as one argument, so file names cannot inject commands. Executable files are
  never run by lazyfinder. Only the user's `config.json` or the built-in `open` default may start a process.
  Config openers that are not found are reported in the status line, not run through a fallback. AC: test with file
  names containing spaces, quotes, `;` and `$(...)`; the stub receives exactly one argument.
- **R-13 GUI vs terminal openers (M1).** An opener may be `"mode": "gui"` (lazyfinder keeps running) or
  `"mode": "terminal"` (lazyfinder suspends, hands the terminal over and restores it on return; for `vim` and similar).
  AC: scenario with a terminal stub; terminal state is restored.

### Preview (Q3)

- **R-14 Space preview popup (M1).** `Space` opens a popup centered over the list showing the preview of the
  selection; `Space` or `Esc` closes it; arrows still move the selection and update the preview, as in Quick Look.
  AC: golden of the popup at 120x40 for text, image and directory fixtures.
- **R-15 Text preview (M1).** Text files show syntax highlighting (chroma), line numbers optional, capped at a size
  limit (default 512 KiB read, configurable) with a "truncated" marker; binary files show type and size instead of
  bytes. AC: fixtures for Go, Markdown, JSON, a binary and a file over the limit.
  M5 (done): binary files show a hex dump, Markdown is rendered, archives list their contents, media shows facts
  and a PDF its first page (ADR 0011). CSV and TSV show as aligned tables, cells cut at 40 cells (ADR 0018). AC:
  `previews-more.json` and the unit tests of `internal/preview`.
- **R-16 Directory preview (M1).** A directory shows its first entries (name, kind, size) and the entry count.
- **R-17 Image preview (M1).** Where the terminal supports the Kitty graphics protocol (detected by query, ADR 0006),
  images render inline in the popup. Elsewhere the popup shows file facts and the key `Q` launches Quick Look
  (`qlmanage -p`) as a separate window (on Linux, which has no Quick Look, the default app through `xdg-open`). AC: unit test of the Kitty escape sequence output; capture in Ghostty of the
  popup; fallback path tested with a stub for `qlmanage`.
- **R-18 Preview never blocks (M1).** Preview loading is asynchronous and cancelable; moving the selection cancels
  the previous load; the UI stays responsive for a 1 GiB file. AC: test with a FIFO and a large sparse file.

### Terminal behavior and quality

- **R-19 Resize (M1).** The layout adapts live to terminal size; the minimum supported size is 60x16, below which a
  "terminal too small" message appears. AC: pty scenario resizing 120x40, 80x24, 60x16, 50x10.
- **R-20 Clean exit (M1).** `q` or `Ctrl+C` leaves the alternate screen, mouse reporting and cursor in their normal
  state, also after a panic. SIGHUP, SIGTERM and SIGINT quit the same way, stopping a paste under way first so it
  leaves no half copy; a program that does not end 10 s later is killed, and a repeated signal ends one that hangs.
  While a terminal opener has the terminal, ^C reaches only that program. AC: scenario checks the restore sequences;
  test with a forced panic; signal tests in cmd/lazyfinder.
- **R-21 No color dependence (M1).** `NO_COLOR` is honored; selection, directory and error states are distinguishable
  without color. AC: golden with `NO_COLOR=1`.
- **R-22 Configuration (M1).** `config.json` is read from `$LAZYFINDER_CONFIG`, else
  `$XDG_CONFIG_HOME/lazyfinder/config.json`, else `~/.config/lazyfinder/config.json`. A missing file means defaults;
  an invalid file is reported with the line and field and the app still starts with defaults. AC: tests for each
  path and for malformed JSON. Schema in ADR 0004.
- **R-23 Isolation (M1).** Tests and scenarios run with a temporary `HOME`, `LAZYFINDER_CONFIG` and fixture trees;
  nothing reads the user's real files. AC: `verificar.sh` runs the suite with `HOME` pointed at a temp dir and a
  check that fails if any test opens a path under the real home.
- **R-24 Read-only (M1).** Milestone 1 never modifies, moves, renames or deletes files. AC: test that no write
  syscalls target the fixture tree (fixture is read-only; the suite passes with it mounted read-only mode `0555`).
- **R-25 Errors (M1, ED-28).** Unreadable directories, broken symlinks and permission errors show a message a user
  can act on in the status line; the app does not crash. AC: fixtures with mode `000` and a dangling symlink.
- **R-26 Logging (M1, ED-28).** Optional JSON log to a file (`--log <path>`), no file names unless `--log-paths`,
  off by default.

### Look, themes, languages and settings (Q8, M1)

The look is Finder's content (columns, sidebar, breadcrumb) in lazymark's style: rounded bordered panels with a
title and a counter, a bar of keys at the bottom and Catppuccin-family palettes. The systems are reused from
lazymark (read-only reference, same author and MIT license): palettes and user themes, `i18n.T` pairs with JSON
catalogs, config persistence, the Kitty module. See `docs/adr/0008-look-themes-languages-settings.md`.

- **R-40 Lazy look (M1).** The sidebar and the list are two rounded panels (focused panel: accent border), the
  list panel has the title `<folder>` and a counter `n of m` in its bottom border, and a bar of keys closes the
  screen (`Label: key | ...`, keys in the accent color). The selected row is a solid bar. AC: goldens at 120x40,
  80x24 and 60x16; real Ghostty capture.
- **R-41 Themes (M1).** `theme` names one of the 15 built-in palettes (default `catppuccin-mocha`; `high-contrast` keeps every text at 7:1), a user theme
  (`<config dir>/themes/<name>.json`, 14 `#rrggbb` colors) or `terminal` (the terminal's own colors). The whole
  screen is painted with the theme unless `screen_background` is `terminal`. An invalid user theme is skipped with
  a warning. `--theme <name>` overrides for one run. AC: unit tests on the loader; scenario with `assert_color`
  checking the selection bar color for two themes.
- **R-42 Languages (M1).** The interface language is `auto` (from `LC_ALL`, `LC_MESSAGES`, `LANG`) or one of
  en, es, pt, fr, de, it, ja, zh (`language` in the config, `--lang`). Every interface text goes through
  `i18n.T`. A missing translation falls back to English. AC: coverage test failing when a text lacks a
  translation in any language (as in lazymark); scenario starting in Spanish.
- **R-43 Settings popup (M1).** `,` (or the gear in the toolbar) opens a popup that changes theme, language,
  image protocol (`auto`/`kitty`/`off`), hidden files, folders first, screen background and mouse with keys or
  clicks; each change applies at once and is saved to `config.json` (atomic write; nothing else in the file is
  lost). AC: unit test of the save; scenario changes the theme and the language and sees the screen change.
- **R-44 Configurable keys (M1).** `keys` in the config maps an action to a list of keys; the defaults of
  `docs/keybindings.md` apply to the rest; a key bound twice or an unknown action is a warning and the default
  stays. The bar of keys and the help show the real keys. AC: unit test; scenario with a rebound key.
- **R-45 Mouse is optional (M1).** `mouse` (default true) and `--no-mouse` turn mouse reporting off (the terminal
  keeps its own selection). AC: the view requests no mouse mode; scenario.
- **R-46 Kitty protocol configuration (M1).** Already R-17: `preview.images` is `auto`, `kitty` or `off`, with
  Quick Look as the fallback; the setting is part of the Settings popup (R-43).

## Requirements for later milestones (summarized)

- **R-30 Live follow (M2, done)** (Q5). Directory listings update without refresh when files appear, change or vanish.
  AC: scenario creates files from another process and sees them within 1 s.
- **R-31 Git consistency (M2, done)** (Q5). A status column or marker (modified, added, untracked, ignored, conflicted)
  and the branch name are shown, refreshed on the same events. Ignored files can be hidden. AC: fixture repo.
- **R-32 Project sidebar mode (M3, done)** (Q5). `lazyfinder --follow <dir>` runs a narrow tree view of one project,
  like the VS Code explorer, highlighting what is new. PROPOSAL for the exact UX.
- **R-33 Disclosure triangles (M3, done)** from the reference screenshot: expand folders in place with `>`.
- **R-34 Search (M3, done)**: the screenshot's Search field; name filter first, content search later.
- **R-35 File operations (M4, done)**: copy, move, rename, trash (never `rm`), new folder, with undo, redo (`U`) and
  a history (`H`).

## Parity with Finder and Explorer search (ADR 0009)

Requirement (2026-10-13): the same features as Finder and as the Windows file search.

- **R-50 Search bar (M3, done).** `/` opens a search bar with the grammar of ADR 0009: words for the name; `kind:`,
  `modified:`, `size:` filters; chips that can be clicked; scope current folder or recursive; results appear while
  typing and can be cancelled with `esc`. AC: unit tests of the parser and the matcher; a tuiqa scenario types a
  query and clicks a chip.
- **R-51 Operations parity (M4, done).** Copy, cut, paste, duplicate, rename, new folder, move to Trash, select many,
  Get Info popup, permissions in octal (`P`, undoable, never through a link). A paste, a duplicate, a move to Trash, an
  undo and a redo run apart from the UI with their progress, and Esc cancels them without a half copy; Get Info counts a folder apart too. AC: fixture
  tests for each; Trash never deletes for good.
- **R-52 Go to folder (M3, done).** `ctrl+g` prompts for a path (with `~` and completion). AC: scenario.
- **R-53 Tabs, tags, saved searches, recents (M6, done).** Tabs, bookmarks, recent folders, quick filter, saved
  searches, tags, Vim-style marks (`m`, `'`), a command palette (`:`) that runs any action by a fuzzy match, custom
  commands, content search and a column layout (ADR 0012). AC: `workspace.json` and the unit tests of `internal/ui` and
  `internal/search`.
- **R-54 Default openers (M1, extended M3, done).** A built-in table of defaults per extension that the config replaces
  (never run anything unconfigured stays true: defaults call `open`). AC: test of the table and an override.
- **R-55 herdr plugin (2026-10-15).** `herdr-plugin.toml` at the root makes the repository a herdr plugin
  (`herdr plugin install MathiasDrizzy/lazyfinder`). Every tab gets one lazyfinder pane in the sidebar mode at the edge
  of `herdr.dock` (Settings: "Sidebar in herdr"), `herdr.columns` columns wide (`herdr.external_columns` when herdr has
  at least `herdr.external_from` columns, an external monitor), without taking the focus. While it runs it goes back to
  that width when herdr changes width (`herdr.adapt`), never stays below `herdr.min_columns`, keeps a width dragged by
  hand otherwise, and `ctrl+b` gives it the width of the normal mode and back (`herdr.fit_normal`); its mode changes
  only with `ctrl+b`, not with its width. A terminal program opens in a new tab of herdr (`herdr.open_in`), which closes
  when only its sidebar is left, and `Space` opens no preview unless `herdr.preview` is on; the
  `toggle` action opens, focuses or closes it, and a sidebar closed with it or with q stays closed in that tab. After
  a restart of herdr a `[[startup]]` hook gives every tab its sidebar back. The pane follows the folder of the focused
  pane of its tab when that folder changes. Outside the plugin nothing changes. AC: unit tests of `internal/herdr`
  against a fake socket, and an isolated herdr that docks it left and right, follows a `cd` and survives a restart
  (ADR 0017).

## Out of scope for M1

Any write operation, live follow, git, tabs, remote filesystems, Windows and Linux support beyond "builds and
tests pass" (macOS is the target; Linux CI runs the portable tests, ED-23).

## Public API (ED-9)

Versioning is `0.y.z` until declared stable. The public surface is: CLI flags, the `config.json` format, and the
exit codes (0 normal, 1 runtime error, 2 usage error). Changes to them are listed in `CHANGELOG.md`.
