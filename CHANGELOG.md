# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) and the project follows
[Semantic Versioning](https://semver.org/). The public API is the command-line interface and `config.json`.

## [Unreleased]

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
