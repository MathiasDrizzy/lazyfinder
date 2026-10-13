# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) and the project follows
[Semantic Versioning](https://semver.org/). The public API is the command-line interface and `config.json`.

## [Unreleased]

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
