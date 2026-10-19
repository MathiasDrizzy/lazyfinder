# Configuration

lazyfinder reads one JSON file. Its location, first match wins:

1. `$LAZYFINDER_CONFIG`
2. `$XDG_CONFIG_HOME/lazyfinder/config.json`
3. `~/.config/lazyfinder/config.json`

No file means the defaults. `lazyfinder --print-default-config` prints them. Unknown keys and wrong types are
errors: the app starts with the defaults and shows the problem in the status bar.

| Key | Meaning | Default |
| --- | --- | --- |
| `version` | Config format; must be `1` | `1` |
| `columns` | Visible columns, in order: `name`, `modified`, `created`, `opened`, `size`, `kind`, `extension`, `permissions`, `owner`, `path`, `tags` | `name`, `modified`, `created`, `size`, `kind` |
| `sort` | `{ "by": column, "descending": bool, "folders_first": bool }` | name, ascending, folders first |
| `show_hidden` | Start with dotfiles visible | `false` |
| `sidebar.favorites`, `sidebar.locations` | Lists of `{ "name", "path" }`; `~` is your home | Finder-like set |
| `openers` | Named programs, see below | none |
| `open_with` | Extension to opener name, e.g. `"txt": "edit"` | none (system `open`, `xdg-open` on Linux) |
| `preview.max_text_bytes` | Bytes read for a text preview | `524288` |
| `preview.images` | `auto` (ask the terminal), `kitty` (force) or `off` | `auto` |
| `preview.markdown` | `render` shows Markdown formatted, `raw` shows the source | `render` |
| `preview.hex_bytes` | Bytes of a binary file shown as a hex dump | `4096` |
| `preview.external_tools` | `auto` lets previews run `mdls`, `pdftoppm` or `sips` (media facts, PDF first page); `off` never runs them | `auto` |
| `theme` | A built-in theme, a user theme (see below) or `terminal` | `catppuccin-mocha` |
| `icons` | `nerd` draws a colored Nerd Font glyph per kind (the herdr-sidebar table: folders open and closed, files by name and extension, apps, links), `emoji` one emoji per kind for fonts without Nerd Font glyphs, `ascii` a plain character, `off` nothing | `nerd` |
| `keybinding_mode` | `dual` (arrows and Vim letters) or `lazy` (arrows only) | `dual` |
| `date_format` | `absolute` (13 Oct 2026, 09:05), `relative` (5 h ago) or `numeric` (2026-10-13 09:05). One format for the whole column; a missing date (empty or before 1980) is `--`. The old `finder` value reads as `absolute`. | `absolute` |
| `icon_size` | Ignored since 0.12 (there is no icon size, [ADR 0015](adr/0015-no-icon-size.md)); old files still load. | |
| `column_separators` | `true` draws a `│` between the column names in the header (not down the rows), and the mouse can drag it to resize | `true` |
| `density` | `compact` (one plain line per row), `normal` (one line, soft stripes and a pill selection) or `comfortable` (two lines, like Finder's roomy rows) | `comfortable` |
| `favorites_height` | lines of the Favorites box; `0` fits it to its folders. Dragging the border between Favorites and Locations, or `ctrl+up` / `ctrl+down`, saves it | `0` |
| `sidebar_width` | cells of the sidebar, 12 to 60; `0` is automatic | `0` |
| `preview_width` | cells of the preview in the column view, 12 to 200; `0` is automatic | `0` |
| `column_widths` | cells of an optional column, 6 to 60, e.g. `{"kind": 24}` | `{}` |
| `icon_map` | extension or whole file name → glyph overrides, e.g. `{"go": "G", "Makefile": "M"}` | `{}` |
| `screen_background` | `theme` paints the whole screen with the theme background; `terminal` keeps the terminal's own | `theme` |
| `popup_background` | `theme` paints popups with the theme background; `terminal` leaves the terminal's own behind them (a translucent terminal stays translucent) | `theme` |
| `confirm_delete` | `false` never asks before moving to the Trash (`u` undoes it); `true` asks as `ops.confirm_trash_above` says | `true` |
| `cd_on_exit` | With the function of `--shell-init` in the shell, quitting leaves the shell in the folder lazyfinder ends in; `false` leaves it where it was (Settings: "Stay in the folder on exit") | `true` |
| `herdr.dock` | Edge of the tab where the [herdr plugin](../README.md#a-sidebar-in-every-herdr-tab) opens the sidebar: `left` or `right` (Settings: "Sidebar in herdr"); sidebars already open stay where they are | `left` |
| `herdr.auto_open` | `true` opens the sidebar by itself in every herdr tab; `false` leaves it to the plugin's Toggle lazyfinder action | `true` |
| `herdr.width` | Width of that sidebar in percent of the tab, 15 to 60 | `25` |
| `language` | `auto` (from `LC_ALL`, `LC_MESSAGES`, `LANG`), `en`, `es`, `pt`, `fr`, `de`, `it`, `ja` or `zh` | `auto` |
| `mouse` | Handle the mouse; `false` leaves it to the terminal | `true` |
| `keys` | Action to list of keys, see [keybindings](keybindings.md) | all defaults |
| `watch.max_dirs` | Folders followed with the operating system's notifications; more are polled | `64` |
| `watch.poll_interval_ms` | Polling period for folders beyond `watch.max_dirs` or that the system cannot watch | `2000` |
| `search.max_results` | Most results a search collects before it stops and says so | `5000` |
| `ops.confirm_trash_above` | Ask before trashing more than this many items at once; `0` asks always | `1` |
| `ops.trash_dir` | Folder that receives trashed items (absolute or `~/…`). A folder `…/Trash/files` is an XDG trash: each item also gets its `.trashinfo` in `…/Trash/info`, so the desktop shows it and can put it back | `~/.Trash` on macOS, `$XDG_DATA_HOME/Trash/files` on Linux (`~/.local/share/Trash/files` when it is not set) |
| `layout` | `list` (detail columns) or `columns` (parent, folder, preview); `M` toggles and saves it | `list` |
| `commands` | Custom commands for the `!` menu, see below | none |
| `search.scope` | where the search looks: `folder`, `subfolders` or `system` (the Spotlight index) | `folder` |
| `search.kinds` | what it looks for: `both`, `files` or `folders` | `both` |
| `search.by` | `name` or `content` (what is inside the files) | `name` |
| `search.content_tool` | `auto` uses ripgrep for `in:` when installed (falls back to a built-in scan), `native` never runs it | `auto` |
| `git.enabled` | Show git state next to each file and the branch in the toolbar | `true` |
| `git.hide_ignored` | Start with files ignored by git hidden (`I` toggles) | `false` |

The Columns row of Settings and the right-click menu of the header offer two presets: **Basic** (Name, Date Modified, Date Created, Size, Kind, the default) and **Full** (every column but Extension, which the name already shows; it can be added from the header menu). What you choose is saved in `columns`.

On narrow windows the optional columns are dropped, the least useful first (Extension, Path, Tags, Owner, Permissions, Last Opened, Date Created, Kind, Size and Date Modified), so the name
stays readable.

## Openers

```json
{
  "version": 1,
  "openers": {
    "edit":  { "argv": ["code", "{file}"], "mode": "gui" },
    "vim":   { "argv": ["nvim", "{file}"], "mode": "terminal" },
    "sheet": { "argv": ["open", "-a", "LibreOffice", "{file}"], "mode": "gui" }
  },
  "open_with": { "txt": "edit", "md": "vim", "xlsx": "sheet" }
}
```

- `argv` is a program and its arguments. It is **never** run through a shell. `{file}` must be a whole element and
  is replaced by the selected path; a path that starts with `-` is passed as `./path`.
- `mode` is `gui` (started in the background) or `terminal` (lazyfinder hands the terminal over until it exits).
- An extension without an entry opens with `open -- <path>` (`xdg-open <path>` on Linux). Nothing else is ever run.
- These openers ship built in, so Open With is useful before any configuration, and `open_with` can name them without
  defining them: `textedit`, `preview`, `vscode` (these three are listed in the menu only when the application is
  installed) and `vim` (terminal). An opener of yours with the same name replaces the built-in one.

## Custom commands

```json
{
  "version": 1,
  "commands": {
    "git status": { "argv": ["git", "status"], "mode": "terminal" },
    "word count": { "argv": ["wc", "-l", "{file}"], "mode": "terminal" },
    "reveal":     { "argv": ["open", "-R", "{file}"], "mode": "gui" }
  }
}
```

`!` lists them; choosing one runs it in the current folder. `{file}` (the selected row) and `{dir}` (the current
folder) are optional, each at most once and always a whole argument; nothing goes through a shell. A command with
`{file}` needs a selection.

## Things lazyfinder remembers

Next to `config.json`: `recents.json` (visited folders), `searches.json` (saved searches), `tags.json` (tags) and
`marks.json` (the folders marked with `m`).
They are plain JSON and safe to delete. Tags are lazyfinder's own, not Finder's colored tags (ADR 0012).

## Themes

`theme` takes one of the 14 palettes of lazymark: `catppuccin-mocha`, `catppuccin-latte`, `catppuccin-frappe`,
`catppuccin-macchiato`, `tokyo-night`, `gruvbox-dark`, `nord`, `dracula`, `one-dark`, `rose-pine`, `kanagawa`,
`everforest-dark`, `solarized-dark` or `solarized-light`; `high-contrast`, white text and light accents on black with
every text at 7:1 at least (WCAG AAA), for low vision; or `terminal`, which uses only the terminal's own 16
colors. `NO_COLOR` turns every color off whatever the theme says; the selection then shows as reverse video.

Your own theme is one file, `<config dir>/themes/<name>.json` (the folder next to `config.json`), with the 14
colors as `#rrggbb`: `base`, `mantle`, `surface0`, `surface1`, `overlay0`, `text`, `subtext0`, `peach`, `mauve`,
`teal`, `green`, `red`, `blue`, `yellow`. The name is the file name (`[a-z0-9._-]+`). Only regular files up to
64 KiB are read; a broken file is skipped and the problem is shown in the status bar at start.

## Language

The interface follows `language`. Names of files and the sidebar entries you configured are never translated. The
names of the file kinds for specific formats ("PNG image") stay in English; the generic ones (Folder, Document,
Alias) are translated. Config error messages are in English.

## Keys

`keys` maps an action to the list of keys that trigger it; naming an action replaces its defaults, the others keep
theirs:

```json
{ "version": 1, "keys": { "quit": ["Z"], "down": ["n", "down"] } }
```

An unknown action, an empty list, or a key that would belong to two actions is a warning shown at start: the
offending entry is ignored and the app still starts. The key bar at the bottom shows the effective keys.

## Settings popup

`,` (or a click on the gear in the top right corner) opens a popup with the theme, language, image protocol,
hidden files, folders first, screen background and mouse. A change applies at once and is written to
`config.json`. Only the changed key is written; everything else in the file (openers, `~` shorthands) is kept, and a
file that is not valid JSON is never overwritten. Run-only options (`--theme`, `--lang`, `--no-mouse`) are not
saved unless you change that same setting in the popup.

## Command line

| Flag | Effect for this run |
| --- | --- |
| `--theme NAME` | Use that theme |
| `--lang CODE` | Use that language |
| `--no-mouse` | Ignore the mouse |
| `--cwd-file FILE` | On exit, write the folder lazyfinder ends in to `FILE` (nothing when `cd_on_exit` is `false`) |
| `--shell-init SHELL` | Print the shell function (`bash`, `zsh` or `fish`) that runs lazyfinder with `--cwd-file` and moves the shell to that folder: `eval "$(lazyfinder --shell-init zsh)"` in `~/.zshrc` |
| `--herdr ACTION` | A step of the herdr plugin, called by `herdr-plugin.toml`: `ensure` docks a sidebar in the tab of the event, `toggle` opens, focuses or closes it. It needs herdr's `HERDR_SOCKET_PATH` |
| `--follow DIR` | Project sidebar mode (`Ctrl+B` toggles it inside the app): a narrow tree of `DIR` with live updates and git state, for a side pane |
