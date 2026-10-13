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
| `columns` | Visible columns, in order: `name`, `modified`, `created`, `size`, `kind` | all five |
| `sort` | `{ "by": column, "descending": bool, "folders_first": bool }` | name, ascending, folders first |
| `show_hidden` | Start with dotfiles visible | `false` |
| `sidebar.favorites`, `sidebar.locations` | Lists of `{ "name", "path" }`; `~` is your home | Finder-like set |
| `openers` | Named programs, see below | none |
| `open_with` | Extension to opener name, e.g. `"txt": "edit"` | none (system `open`) |
| `preview.max_text_bytes` | Bytes read for a text preview | `524288` |
| `preview.images` | `auto` (ask the terminal), `kitty` (force) or `off` | `auto` |
| `theme` | A built-in theme, a user theme (see below) or `terminal` | `catppuccin-mocha` |
| `screen_background` | `theme` paints the whole screen with the theme background; `terminal` keeps the terminal's own | `theme` |
| `language` | `auto` (from `LC_ALL`, `LC_MESSAGES`, `LANG`), `en`, `es`, `pt`, `fr`, `de`, `it`, `ja` or `zh` | `auto` |
| `mouse` | Handle the mouse; `false` leaves it to the terminal | `true` |
| `keys` | Action to list of keys, see [keybindings](keybindings.md) | all defaults |
| `watch.max_dirs` | Folders followed with the operating system's notifications; more are polled | `64` |
| `watch.poll_interval_ms` | Polling period for folders beyond `watch.max_dirs` or that the system cannot watch | `2000` |
| `search.max_results` | Most results a search collects before it stops and says so | `5000` |
| `ops.confirm_trash_above` | Ask before trashing more than this many items at once; `0` asks always | `1` |
| `ops.trash_dir` | Folder that receives trashed items (absolute or `~/…`) | `~/.Trash` |
| `git.enabled` | Show git state next to each file and the branch in the toolbar | `true` |
| `git.hide_ignored` | Start with files ignored by git hidden (`I` toggles) | `false` |

On narrow windows the Date Created, Kind, Size and Date Modified columns are dropped, in that order, so the name
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
- An extension without an entry opens with `open -- <path>`. Nothing else is ever run.
- These openers ship built in, so Open With is useful before any configuration, and `open_with` can name them without
  defining them: `textedit`, `preview`, `vscode` (these three are listed in the menu only when the application is
  installed) and `vim` (terminal). An opener of yours with the same name replaces the built-in one.

## Themes

`theme` takes one of the 14 palettes of lazymark: `catppuccin-mocha`, `catppuccin-latte`, `catppuccin-frappe`,
`catppuccin-macchiato`, `tokyo-night`, `gruvbox-dark`, `nord`, `dracula`, `one-dark`, `rose-pine`, `kanagawa`,
`everforest-dark`, `solarized-dark` or `solarized-light`; or `terminal`, which uses only the terminal's own 16
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
| `--follow DIR` | Project sidebar mode: a narrow tree of `DIR` with live updates and git state, for a side pane |
