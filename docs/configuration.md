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
| `theme` | `auto`, `dark` or `light` | `auto` |
| `watch` | Reserved for live follow (milestone 2) | see the defaults |

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
