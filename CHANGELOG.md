# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) and the project follows
[Semantic Versioning](https://semver.org/). The public API is the command-line interface and `config.json`.

## [Unreleased]

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
