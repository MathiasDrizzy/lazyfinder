<div align="center">

# lazyfinder

**A lazy file manager for the terminal, like Finder, without leaving it.**

Columns, a sidebar, a Space preview popup and open-with, with keyboard and mouse.

[![CI](https://github.com/MathiasDrizzy/lazyfinder/actions/workflows/ci.yml/badge.svg)](https://github.com/MathiasDrizzy/lazyfinder/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/MathiasDrizzy/lazyfinder)](go.mod)
[![License: MIT](https://img.shields.io/github/license/MathiasDrizzy/lazyfinder)](LICENSE)

<img src="docs/screenshots/list.png" alt="lazyfinder with rounded panels, a sidebar, the columns Name, Date Modified, Date Created, Size and Kind, and a key bar" width="100%">

[Install](#install) · [Quick start](#quick-start) · [Keys](#keys) · [Configuration](#configuration) · [Roadmap](docs/ROADMAP.md)

</div>

## Why lazyfinder

- **Looks like Finder.** A sidebar of Favorites and Locations, sortable columns, folders marked with `>`, a
  breadcrumb at the bottom and back and forward buttons.
- **Keyboard or mouse.** Click to select, double click to enter or open, wheel to scroll; every action also has a key.
- **Open with what you choose.** A `.txt` in your editor, a `.xlsx` in Excel or LibreOffice, a photo in Preview.
  Nothing runs unless you configured it.
- **Space shows what is inside.** Highlighted text, images (Kitty graphics protocol), folder contents, and Quick
  Look when your terminal cannot draw images.
- **Read-only and safe.** Version 0.x never changes your files. Programs start without a shell, so file names
  cannot inject commands.

## What it does

### Preview with Space

Press `Space` on a file or folder for a popup, and `↑`/`↓` to look at the next one without closing it.

Text is highlighted, Markdown is rendered, zip and tar archives list their contents (nothing is extracted), binary
files show a hex dump, audio and video show their duration and codecs, and a PDF shows its first page where the
terminal supports Kitty images. `preview.external_tools` turns off the helper programs.

<img src="docs/screenshots/preview-text.png" alt="Space preview of a Go file with syntax highlighting" width="100%">

<img src="docs/screenshots/preview-image.png" alt="Space preview of an image drawn inside the popup" width="100%">

### Open with

`Enter` opens the selected file with the program set for its extension (the system `open` by default). `O` lists
every configured program and lets you pick one.

<img src="docs/screenshots/open-with.png" alt="The Open With menu listing a configured editor and the system default" width="100%">

## Install

Requires macOS or Linux and a terminal with mouse support. Images need a terminal that speaks the Kitty graphics
protocol (Ghostty, kitty, WezTerm).

```sh
go install github.com/MathiasDrizzy/lazyfinder/cmd/lazyfinder@latest
```

Or build from a clone:

```sh
go build -o bin/lazyfinder ./cmd/lazyfinder
```

Shell completions and the manual page:

```sh
lazyfinder --completions zsh > "${fpath[1]}/_lazyfinder"   # or bash, or fish
man ./docs/lazyfinder.1
```

A Homebrew formula for a tap is in [packaging/homebrew](packaging/homebrew/lazyfinder.rb).

## Quick start

```sh
lazyfinder            # starts in the current folder
lazyfinder ~/Projects # starts in a folder
lazyfinder --print-default-config > ~/.config/lazyfinder/config.json
```

## Keys

| Key | Action |
| --- | --- |
| `↑` `↓` / `k` `j` | Move the selection |
| `Enter` / `→` / `l` | Enter a folder or open a file |
| `Backspace` / `←` / `h` | Go to the parent folder |
| `Alt+←` `Alt+→` / `[` `]` | Back and forward in history |
| `Space` | Preview popup; `Esc` or `Space` closes it |
| `O` | Open With menu |
| `Q` | Quick Look (in the preview popup, macOS) |
| `s` / `S` | Sort by the next column / reverse the order |
| `.` | Show or hide dotfiles |
| `r` | Reload the folder |
| `Tab` | Move between the sidebar and the list |
| `,` | Settings popup (theme, language, images, ...) |
| `q` | Quit |

Every key can be rebound with `keys` in `config.json`. The mouse does the same: click rows, headers, sidebar entries, breadcrumb segments and the back and forward
buttons. The full list is in [docs/keybindings.md](docs/keybindings.md).

## Configuration

The config is one JSON file: `$LAZYFINDER_CONFIG`, else `$XDG_CONFIG_HOME/lazyfinder/config.json`, else
`~/.config/lazyfinder/config.json`. Without a file the defaults apply. An invalid file starts the app with the
defaults and shows what is wrong. See [docs/configuration.md](docs/configuration.md).

## File operations

Copy `c`, cut `x`, paste `p`, duplicate `D`, rename `F2`, new folder `N`, Trash `d` and undo `u`, on the selected row or on
the rows you marked with `v`, `ctrl+a` or ctrl-click. Nothing deletes or overwrites for good: the Trash is a folder, a
taken name gets a `copy` suffix and every step can be undone (see [ADR 0010](docs/adr/0010-file-operations.md)). `i` shows Get Info.

![Marked rows and the Trash confirmation](docs/screenshots/marks.png)

## Sidebar mode

![Sidebar mode at 36 columns](docs/screenshots/sidebar-mode.png)

A narrow panel meant to sit next to another terminal, like the VS Code explorer and the herdr sidebar. Start it with
`lazyfinder --sidebar [dir]` (`--follow <dir>` is the same), with `Ctrl+B` in any window, or from Settings (Side mode).
The window and the sidebar mode also switch by themselves when you resize the terminal: below 60 columns the window
becomes the sidebar, from 80 columns up the sidebar becomes the window again. It works from 24 columns:

- clickable icon-only tabs **Files**, **Search** and **Git** (a tooltip names each), like the herdr sidebar;
- action icons at the right of the project name: new file, new folder, collapse all (the list updates by itself, so there is no refresh button). Every icon has a
  tooltip in the bottom row, and the row or button under the pointer is painted;
- the project name in capitals, the tree with the git letter and color at the right edge (`M` modified, `U` new, `A`
  added, `D` deleted, `R` renamed, `!` conflict) and, at the bottom, the branch with how far it is from its upstream;
- it updates by itself: files created, changed or removed by another program appear at once, marked with `+`;
- the **Git tab** works like the source control view of the herdr sidebar: staged and unstaged changes with their
  counts, stage, unstage and discard (asking first) by key or button, the diff of a file, stashes with pop and drop,
  fetch, pull and push with how far you are from the upstream, and a branch menu with search, create and a warning
  about changes you have not saved. It never commits, rebases or merges: `L` opens lazygit for that.

![The Git tab, dark and light](docs/screenshots/git-tab-dark.png)
![The Git tab in a light theme](docs/screenshots/git-tab-light.png)
![The selected row shows its buttons: open changes, discard, stage](docs/screenshots/git-tab-buttons.png)
![The branch menu](docs/screenshots/git-branch-menu.png)

![Project mode with git letters](docs/screenshots/project-mode.png)

## Search

![Search with filter chips](docs/screenshots/search.png)

Press `/` to search the folder or its subfolders by name, kind, size and date, with clickable filter chips. See [search](docs/search.md).

## Tabs, bookmarks, tags and more

`ctrl+t` opens a tab, `b` bookmarks a folder in the sidebar, `E` lists recent folders, `f` filters the folder by name,
`t` tags files and `F` runs saved searches. `!` runs your own commands from `config.json`, `in:text` searches inside
files (with ripgrep when installed) and `M` switches to a column view with the parent, the folder and a preview.
See [keybindings](docs/keybindings.md) and [configuration](docs/configuration.md).

<img src="docs/screenshots/columns.png" alt="The column layout: parent folder, current folder and a preview of the selection" width="100%">

## Live follow and git

The open folder updates by itself and shows git state and the branch. In the normal window the branch is in the toolbar
and the footer; click it, or press `B`, to switch branches, and right click a changed file to see, stage, unstage or
discard its changes. Moving past the last row goes to the first.

![Live follow and git state](docs/screenshots/live-git.png)

## Look, themes and languages

lazyfinder looks like Finder in what it shows and like [lazymark](https://github.com/MathiasDrizzy/lazymark) in how
it draws it: rounded panels, the accent on the focused one and a key bar at the bottom. It ships the 14 lazymark
themes (`catppuccin-mocha` by default), takes your own from `<config dir>/themes`, and speaks English, Spanish,
Portuguese, French, German, Italian, Japanese and Chinese. Press `,` (or click the gear) to change the theme,
language, image protocol and more; changes apply at once and are saved. For one run only:
`lazyfinder --theme nord --lang es --no-mouse`.

<img src="docs/screenshots/settings.png" alt="The settings popup over the list" width="100%">

Icons are colored Nerd Font glyphs, classified like the [herdr sidebar](https://github.com/alexarthurs/herdr-sidebar) plugin does (`icons` in Settings: `nerd`, `emoji`, `ascii` or `off`; `icon_map` overrides an extension or a file name). There is no icon size: a glyph cannot grow inside a terminal cell ([ADR 0015](docs/adr/0015-no-icon-size.md)). A `│` separates the column names in the header like in
Finder (`column_separators`), and **Columns** in Settings or a right click on the header shows fewer or more data per
file, with a *Basic* and a *Full* preset.

![The column menu](docs/screenshots/columns-menu.png)

The sidebar is two boxes, Favorites and Locations, and every entry has its own icon like in Finder:

![Sidebar with an icon per entry](docs/screenshots/sidebar-icons.png)

<img src="docs/screenshots/look-dark-small.png" alt="Small icons, dark theme" width="100%">

<img src="docs/screenshots/look-dark-large.png" alt="Large icons, dark theme" width="100%">

<img src="docs/screenshots/look-light-small.png" alt="Small icons, light theme" width="100%">

<img src="docs/screenshots/look-light-large.png" alt="Large icons, light theme" width="100%">

<img src="docs/screenshots/theme-latte.png" alt="The catppuccin-latte theme with a preview popup" width="100%">

<img src="docs/screenshots/spanish.png" alt="The interface in Spanish" width="100%">

## Build and test

```sh
go vet ./...
go test -race ./...
scripts/tuiqa.sh   # end-to-end scenarios against the real binary in a pty
```

Design decisions are in [docs/adr](docs/adr) and the requirements in [docs/SPEC.md](docs/SPEC.md).

## License

[MIT](LICENSE)
