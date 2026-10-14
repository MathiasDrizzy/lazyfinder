# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) and the project follows
[Semantic Versioning](https://semver.org/). The public API is the command-line interface and `config.json`.

## [Unreleased]

### Added

- Homebrew: `brew install mathiasdrizzy/tap/lazyfinder` (macOS). Each release publishes the cask to the tap.
- Issue and pull request templates.

### Changed

- `config.json` and the files of recent items, saved searches and tags are written readable only by you (0600, in a
  0700 folder).
- `golang.org/x/net` is v0.60.0 (none of the advisories in v0.39.0 was reachable from lazyfinder).

### Fixed

- On a narrow window the columns cut a long title, and then a long kind, before a whole column goes away: in Spanish
  at 120 cells the date created no longer disappears. Dates are never cut.

### Removed

- The Homebrew formula in `packaging/homebrew`: the cask in the tap replaces it.
- Internal planning notes and review screenshots from `docs/`.

### Security

- Looking at a git repository no longer runs programs from that repository's own `.git/config`: the status and the
  diff switch off its `core.fsmonitor`, its clean, smudge and process filters and its textconv drivers. The same
  settings from your global config still apply.
- Escape sequences in a file name, an archive entry or a previewed file no longer reach the terminal: the preview
  shows control characters as their pictures (ESC as `␛`), and the screen drops any OSC, DCS or APC sequence, such as
  an OSC 8 link. A line break in a name no longer breaks the row.
- A FIFO, `/dev/tty`, `/dev/zero` or a link to one in a folder no longer freezes lazyfinder: the column view, the
  preview, the archive list, the content search and the image preview open only regular files, without blocking.

## [0.12.1] - 2026-10-14

### Added

- Quitting leaves the shell in the folder lazyfinder ends in: `eval "$(lazyfinder --shell-init zsh)"` in `~/.zshrc`
  (or bash, or fish) makes `lazyfinder` a shell function that runs it with the new `--cwd-file` flag and moves to
  that folder. Settings → "Stay in the folder on exit" (`cd_on_exit`) turns it off.

### Changed

- The tree of the sidebar mode never lists `.git`, even with hidden files shown, as the herdr sidebar does; the normal window still shows it.
- The Git tab of the sidebar mode only shows what changed: the changes, the stashes, the branch and the diff of a file. It no longer stages, unstages, discards, stashes, switches branches, fetches, pulls or pushes, and it has no git buttons in its header; `L` opens lazygit for that. The normal window keeps its branch menu and the git actions of its right-click menu.
- The buttons of the top rows are the three-cell chips of the herdr sidebar (` g `), the action buttons side by side
  and one free cell between groups: the four-cell boxes one cell apart of 0.12.0 were too big and too far apart, and
  a Nerd Font glyph, drawn wider than its cell, now sits in the middle of its box.
- The box the pointer lights on a button of a Nerd Font glyph is sized from the glyph's measured ink, with a block
  of eighths of a cell on its left side, so the glyph sits in its middle in Ghostty (margins at most 2 pixels apart)
  and every box is as wide as the others (43 to 45 pixels, the collapse box was 57); the mode button and the gear
  sit side by side like the action buttons.
- The Basic preset of columns is the default one and shows the date created; the Full preset leaves out the
  extension, which the name already shows (it can still be added from the header menu), and on a narrow window the
  extension is the first column to go.

### Fixed

- In a git repository a file that appears while lazyfinder is open shows its git state, not a + that hid it: an untracked file says U and a committed one carries no mark, as `git status` does. The + stays for new files outside a repository or ignored by git.
- With `"screen_background": "terminal"`, the left side of the box lit on a button showed a white strip: the box
  is drawn with blocks in its own color, never with reversed colors.
- In the sidebar, the tall box lit on the mode button and the gear had a notch on its left side: its rows above and
  below now start at the same half cell as the middle row.
- A test of the path bar failed about one run in four, depending on the name of the temporary folder: it now finds
  the segment by its position.

## [0.12.0] - 2026-10-14

_Parity with the herdr sidebar is tabulated in `docs/PARITY-herdr-sidebar.md`._

### Added

- A new folder or file is made at once with a free name (`untitled folder`, `untitled 2`, …) and its row shows the
  name selected, as in the Finder: typing replaces it, enter keeps it, esc takes the new item back. The prompt names
  what is being made, with its icon.

- The sidebar mode is drawn like the herdr sidebar (Mathias compared both, 2026-10-13): no frame of its own, a tab
  bar three rows tall whose active and hovered chips reach the rows above and below with half blocks (each row takes
  the click), four-cell tab chips that center the wide Nerd Font glyphs, dimmed tree buttons, the project name in
  blue, and tree rows with the git letter (or a dot for a folder) two cells from the edge. The rows wear the colors
  of the normal window, `>`/`v` before a folder, blue folder icons and names, so both modes look like one program. The selected row keeps the colors of its icon and name.

- The tree buttons of the sidebar mode (new file, new folder, collapse all) show while the pointer moves over the pane
  and fade three seconds after it stops, like the title bar buttons of the herdr sidebar; the border of the panel no
  longer repeats the project name that heads the tree.

- The window and the sidebar mode switch by themselves when the terminal is resized: narrower than 60 columns the
  window becomes the sidebar, and from 62 columns up the sidebar becomes the window again (the two-cell gap keeps
  a drag along the limit from flipping it on every cell). The mode the program started in stays until the first resize across a limit.

- The Git tab of the sidebar mode, modelled on the herdr sidebar: Staged Changes, Changes and Stashes sections with
  counts; stage, unstage and discard (with a confirmation) by key or by the buttons of the row; the diff of a file in a
  popup; pop and drop for stashes; fetch, pull (fast-forward only) and push with ahead and behind; conflicts marked
  and protected; a branch menu with search, create and a warning about unsaved changes (`B`, or click the branch in
  the toolbar or the footer); `L` opens lazygit. Commit, rebase and merge are left out on purpose.
- Right click on a changed file in the normal window: Show changes, Stage, Unstage, Discard changes.
- Buttons that appear on the selected row and on the row under the pointer, hover painting and a tooltip in the bottom
  row for every icon and button; navigation wraps around at the first and last row.
- `docs/qa-matriz.md`, the QA matrix: every function with the test or scenario that proves it, checked by
  `TestQAMatrixNamesExist`; a random sweep of keys, clicks, hover and resizes in both modes (`LAZYFINDER_SWEEP=<n>`
  widens it) and exploratory tests for odd names, broken links, unreadable folders and a repository in each state.

- Sidebar mode is a mode of its own: `--sidebar [dir]`, a Settings row and `Ctrl+B`, clickable Files, Search and Git tabs
  (the Git tab lists the changed files), the project name in capitals, the git letter at the right edge and the branch
  with ↓behind ↑ahead at the bottom, one line per row whatever the density.
- Icons follow the herdr sidebar table: a colored glyph per file kind (by whole name and extension), the same in the
  list and the tree, an `emoji` icon style for fonts without Nerd Font glyphs, and `icon_map` accepts whole file names.
  The table is MIT licensed (see `NOTICE`).

### Changed

- Every button of the top rows (actions, Search, mode, Settings) is a box centered on its glyph, one free cell from
  the next, in the same dimmed color in the window and the sidebar mode; the pointer lights it with the theme's
  surface color and the active tab a step lighter, like the keycaps of the herdr sidebar.
- The file icons take the theme color closest to the one the herdr sidebar gives each kind, so they change with the
  theme like everything else; no color is written outside the theme package any more.

- Releases carry a GitHub build provenance attestation (`actions/attest`) for every archive and SBOM instead of a
  cosign signature of `checksums.txt`; verify with `gh attestation verify <file> -R MathiasDrizzy/lazyfinder`.
- The default row density is now wide, closer to Finder: two-line rows, a soft stripe on every second row and a
  selection pill with one cell of margin on each side. `density` accepts `compact` (one plain line), `normal` (one
  line with stripes and pill) and `comfortable` (wide, the default).
- Search: the filters moved to their own row above the prompt as labelled buttons (`Look in`, `What`, `By`, with
  their keys); quick filters that do not fit are dropped whole, never cut; the operators (`kind:`, `ext:`, `size:`,
  `modified:`, `created:`, `in:`) are explained in the help, and a click on the field gives it the keyboard back.
- The Favorites and Locations boxes fit their content and leave the rest of the column free; drag the border between
  them, or press `ctrl+up` / `ctrl+down`, to resize Favorites (saved as `favorites_height`); a double click fits it
  again. This replaces the fixed 50/50 split.
- Dates use one format for the whole column. The default is the absolute date with its time (`13 Oct 2026, 09:05`), never
  "Yesterday" mixed with dates; `relative` (`5 h ago`) is a choice and then applies to every row. The old `finder`
  value reads as `absolute`.
- Settings no longer lists "Images (Kitty)" or "Mouse", like lazymark: the terminal is asked for Kitty support by
  itself (`preview.images` stays in the file to force it off) and the mouse is `mouse` in the file or `--no-mouse`.

### Fixed

- The tooltip of the new folder icon no longer covers the name prompt (it read as a stray "N").

- A detached HEAD shows where it is, `HEAD (detached at 1a2b3c4)`, in the Git tab and the bottom bar.
- Outside a repository the Git tab shows no branch, fetch, pull or push buttons.
- With a merge conflict, discarding or staging everything names the files in conflict and waits for a yes (staging
  marks a conflict as solved).
- The buttons of the Git tab header were drawn one cell left of their click area.
- The end-to-end scenarios wait for what they check instead of reading the screen at once: a check right after a git
  command failed two runs in six with scenarios in parallel. A test rejects a check right after a git key.

- The last visible row of the list did not take the click: the click area ended one row before the bottom border.

- The buttons of the toolbar, the tree header and the rows of the Git tab are all three cells wide with the glyph in
  the middle, so the hover and the click cover the cells they show; the sidebar tabs are icon-only square buttons with
  a tooltip, the refresh button is gone (the list updates by itself), the Search lens is the Nerd Font one, and the
  new file, new folder and collapse glyphs are the codicons of the herdr sidebar.
- The Settings button, the mode button, the Search field and the tree buttons were drawn two cells away from the
  cells that the click and the hover used (the gear took two cells on the screen and four in the zones); every zone of
  the top row now covers exactly what is drawn, and a test checks it at 14 widths in both modes.
- `.gitignore` and `.gitattributes` keep the red of the git icon instead of being dimmed; a click anywhere on a folder
  row of the sidebar tree opens or closes it.
- Get Info and other popups were 58 cells wide even on a 20-column terminal.
- The hover area of a search chip that a narrow terminal cuts off ran past the edge of the screen.
- A popup drawn over a line with wide characters (Chinese, Japanese) pushed the line one cell past the terminal.
- The row counter of the Git tab counted the files of the Files tab.

### Removed

- `icon_size` and its Settings row: a glyph cannot grow inside a terminal cell, so the sizes only added blank lines and
  a box around the icon and the gear ([ADR 0015](docs/adr/0015-no-icon-size.md)). The sidebar keeps its icons. Old
  files that still have the key load as before.

### Fixed

- A JPEG with a C2PA/JUMBF manifest (the ones AI tools and cameras write) was shown as a hex dump; it is a picture
  now. The header is read from the file, not from the first 8 KiB. WebP, BMP and TIFF are decoded too, and a picture
  that cannot be drawn (HEIC, a file over 25 MB) is described in words, never dumped as hex.
- A date that is empty or from before 1980 (a system folder whose birth time is 0, shown as `31/12/1969`) is now `--`.

## [0.11.0] - 2026-10-13

### Added

- Favorites and Locations are two boxes in the sidebar, each with its own border and title (the one holding the
  selection takes the focus color), like Folders, Tasks and Tags in lazymark. Each starts at half the sidebar;
  Favorites stretches by itself as folders are added (Locations keeps four rows) and shrinks back when they are
  removed; what does not fit scrolls inside its box.
- Every Favorites and Locations entry has its own Nerd Font icon in the accent color, like Finder: clock for Recents,
  house, Desktop, Documents, Downloads, Applications, Music, Pictures, Movies, Trash and disks. They are chosen by
  where the entry points (a translated name keeps its icon) and show at every `icon_size`.
- A `│` between the column names in the header, like Finder (not down the rows); `column_separators` and a Settings row
  switch it off, and dragging it still resizes the column.
- `icon_size` (`small`, `medium`, `large`) in Settings and `config.json`: more air around the icons and icons in the
  sidebar from `medium`, every icon on a tile with two-line rows in `large`, and a Nerd Font gear on a raised tile
  that grows with it, with a click zone as wide as the tile ([ADR 0014](docs/adr/0014-icon-size-setting.md)).
- Owner and Path columns; **Basic** (Finder's four columns) and **Full** presets in the header menu and in a Settings
  row, saved in `columns`.
- Project mode from inside the app: `Ctrl+B` turns the window into the `--follow` tree and back, keeping your columns.
- Git letters like the VS Code explorer, in color: `M` modified, `U` untracked, `A` added, `D` deleted, `R` renamed,
  `!` in conflict, `I` ignored. They used to be `?` for untracked and `!` for ignored.
- Git state now refreshes by itself when another program stages, commits or switches branch (the `.git` folder is
  watched). A new file shows up in about 100 ms even in a folder of 5000 files (tuiqa `sidebar-git`).

### Changed

- Project mode no longer opens ignored folders such as `node_modules` by itself.

### Fixed

- A data race in the file watcher: closing it while a change was being delivered could send on a closed channel.
- On a light theme the plain text of the list (file names, dates) took the terminal's own light foreground and was
  almost invisible; it now takes the theme's text color.

## [0.10.0] - 2026-10-13

Includes what 0.9.0 added (search field, texts, mouse resizing), which had no entry.

### Added

- A **Search** field at the top right, like Finder's: a click or `/` focuses it and it shows what you type.
- Search choices: where (this folder, subfolders, the whole system through the Spotlight index), what (files and
  folders, files, folders) and how (by name or by content), on `Tab`, `Ctrl+K` and `Ctrl+N`, as chips, in Settings
  and in `search.scope`, `search.kinds` and `search.by`; they are remembered.
- Resize with the mouse: drag the sidebar border, the preview border of the column view or the right edge of a
  column header; a double click fits the content. `<` `>` and `Ctrl+←` `Ctrl+→` do the same from the keyboard, and
  `sidebar_width`, `preview_width` and `column_widths` remember the sizes.
- Favorites: right click a folder to add it or a favorite to open, rename, move or remove it; in the sidebar `d`
  removes, `F2` renames and `K` `J` reorder; the Trash is a place in Locations.
- `?` or `F1` opens a Help popup with the keys and gestures; Settings has a Keybindings row (Lazy / Lazy + Vim) and
  `keybinding_mode` saves it.
- Columns like Finder's: Date Last Opened, Extension, Permissions and Tags join the old ones; a right click on the
  header chooses and moves them; `date_format` and `density` (comfortable rows are two lines tall) are in Settings and
  `config.json`.
- **Recents** in the sidebar lists the last folders; `Ctrl+U` and `Ctrl+D` page like in lazymark.

### Changed

- Settings shows the date format and the row density in the interface language.
- A scenario at 120 columns in Spanish checks that the default columns are read whole and that the selected row
  is blue-gray in the list, the column view and the sidebar.
- The selection bar is the theme's blue-gray surface, like lazymark's, instead of salmon.
- Popups close only with `Esc`: a click outside does nothing. In Settings the first click selects a row and the second
  changes it, and a hint line says so. The key hints of the bottom row run their action when clicked.
- A wrong value in `config.json` no longer throws the whole file away: that key goes back to its default, a warning
  says which, and the rest is kept (a file that is not JSON, or with a wrong `version`, still fails).
- `popup_background` and `confirm_delete` are in Settings and `config.json`.
- Columns that have no width in `column_widths` fit their title and their widest cell, so Kind and Date Modified are
  no longer cut when the window has room.
- Settings write yes and no like lazymark (they said on and off).

## [0.8.0] - 2026-10-13

### Added

- Nerd Font icons per kind with a colour each (`icons`: nerd, ascii, off; `icon_map` overrides an extension) and an
  Icons row in Settings.
- `docs/PARITY-lazymark.md`, the audit of what lazymark has that lazyfinder should share.
- Releases get an SPDX SBOM per archive and a keyless cosign signature of `checksums.txt`.

### Changed

- Settings and messages are worded like lazymark (`terminal`, `tema`, `auto`), and the sidebar, the title and the
  breadcrumb show the standard places in the interface language.
- The kind of each file (Folder, Excel Spreadsheet, Go Source, ...) and the settings errors are translated into the
  eight interface languages. In narrow columns a long translated kind is cut with an ellipsis; Get Info shows it whole.

### Fixed

- Text that was hard to read in the light themes, nord, one-dark and catppuccin-frappe (dim text, titles, the
  selected row) is lifted to at least 3:1 contrast, keeping each palette's hue.

## [0.7.0] - 2026-10-13

### Added

- `--completions bash|zsh|fish` prints a completion script.
- A manual page (`docs/lazyfinder.1`), shipped in the release archives.
- A Homebrew formula in `packaging/homebrew/`, ready for a tap.
- A test that moves a folder between two real volumes (a disk image) and undoes it.

## [0.6.0] - 2026-10-13

### Added

- Tabs (`ctrl+t`, `ctrl+w`, `}`, `{`), bookmarks (`b`), recent folders (`E`), a quick filter by name or glob (`f`),
  saved searches (`ctrl+s` in the search bar, `F`), tags (`t`, `T`, `tag:`) and custom commands (`!`, `commands`).
- Content search with `in:`; it uses ripgrep when installed (`search.content_tool`) and a built-in scan otherwise.
- A column layout (parent, folder, preview), toggled with `M` and set with `layout`.
- ADR 0012. Settings `layout` and `search.content_tool`.

## [0.5.0] - 2026-10-13

### Added

- Hex view for binary files, replacing the "Binary file" message (`preview.hex_bytes`).
- Archive listing for zip, tar and tar.gz files; nothing is extracted and hostile names cannot inject terminal codes.
- Markdown files are shown rendered (`preview.markdown` to see the source).
- Duration, dimensions and codecs of audio and video files, and the first page of a PDF as an image, through
  `mdls`, `pdftoppm` or `sips` when present (`preview.external_tools` turns them off).
- ADR 0011 records why Sixel and iTerm2 images are not part of this release.

## [0.4.0] - 2026-10-13

### Added

- File operations, none of which deletes or overwrites for good (ADR 0010): copy (`c`), cut (`x`) and paste (`p`),
  duplicate (`D`), rename (`F2`), new folder (`N`) and move to the Trash (`d` or `Delete`), each with undo (`u`).
  Names that are taken get Finder's `copy` suffix. Trashing a batch asks first (`ops.confirm_trash_above`).
- Selection of many: `v` marks and moves down, `ctrl+a` marks all, ctrl-click marks with the mouse, `esc` clears.
- Get Info popup (`i`) with kind, location, size (the content of a folder), modification time and permissions.
- Settings `ops.confirm_trash_above` and `ops.trash_dir`.

### Changed

- `x`, `c`, `p`, `d`, `u`, `i`, `v`, `D`, `R` and `N` are default keys now. A `keys` entry that gives one of them to
  another action is reported as a clash at start and the default stays.

## [0.3.0] - 2026-10-13

### Added

- `lazyfinder --follow <dir>`: a narrow tree of one project for a side pane (works from 24x8 up): no sidebar, only
  names, the first level open, `Enter` opens and closes folders instead of leaving the project, a folder that
  appears is shown open, and everything new keeps its `+` and green name for the whole session.
- Folders open in place like Finder's list view: `+` or a click on the `>` marker (it becomes `v`) shows the children
  indented under the folder, `-` closes it. Open folders are followed live and remembered while you move around.
- Go to folder (`ctrl+g`): type a path (`~`, relative or absolute), `Tab` completes folder names, `Enter` goes there
  (a file path goes to its folder and selects it).
- Built-in openers for Open With: TextEdit, Preview, Visual Studio Code (when installed) and Vim. They can be mapped
  to extensions in `open_with` without defining them, and a user opener with the same name replaces them.
- Search bar (`/`, `ctrl+f`, or the lens in the toolbar): words match the name (substrings or `*`/`?` globs, quotes
  for phrases) and `kind:`, `ext:`, `size:`, `modified:`, `created:` filters narrow it, with clickable chips for
  images, text, folders, today and this week. Results appear while typing, in the folder or in its subfolders
  (`Tab`, streamed in the background, `esc` cancels, `search.max_results` caps it). Each result shows where it lives.

## [0.2.0] - 2026-10-13

### Added

- Live follow: the folder you are looking at updates on its own when files appear, change or vanish (fsnotify on the
  visible folder, polling as a fallback, 50 ms debounce). New files carry a `+` and a green name until you leave the
  folder.
- Git: a state letter next to each file inside a repository (`M` modified, `A` added, `R` renamed, `D` deleted,
  `U` conflicted, `?` untracked, `!` ignored; a folder shows the strongest state inside it), the branch with its
  ahead/behind counts in the toolbar, and `I` to hide ignored files. Configurable under `git` and `watch`.

## [0.1.0] - 2026-10-13

### Added

- Finder-like file list with the columns Name, Date Modified, Date Created, Size and Kind, sortable from the
  header or with `s`/`S`, with folders first and a `>` marker on folders.
- Keyboard and mouse navigation: click to select, double click to enter or open, wheel scrolling, back and forward
  buttons and history, and a clickable breadcrumb.
- Sidebar with Favorites and Locations read from `config.json`.
- Open With: an opener per file extension, the system `open` by default, programs run without a shell and only
  when configured; terminal programs get the terminal while they run.
- Space preview popup: highlighted text, images through the Kitty graphics protocol (Ghostty, kitty, WezTerm),
  folders, and Quick Look (`Q`) as the fallback on macOS.
- Hidden files toggle (`.`), reload (`r`), `NO_COLOR` support and a minimum-size message.
- `lazyfinder --version`, `--print-default-config`, `--log` and `--log-paths`.
- Lazy look from lazymark: rounded panels (the focused one has the accent border), a `n of m` counter, the
  folder in the panel title and a key bar generated from the effective keys.
- Themes: the 14 lazymark palettes, your own in `<config dir>/themes/<name>.json`, `terminal` (the terminal's own
  colors), `screen_background` and `--theme`.
- Languages: English, Spanish, Portuguese, French, German, Italian, Japanese and Chinese (`language`, `--lang`,
  detected from the locale by default).
- Settings popup (`,` or the gear) that applies changes at once and saves them to `config.json` without touching
  the rest of the file.
- Configurable `keys` and `mouse` (`--no-mouse`).
