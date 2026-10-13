# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) and the project follows
[Semantic Versioning](https://semver.org/). The public API is the command-line interface and `config.json`.

## [Unreleased]

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
