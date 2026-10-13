# `config.json` schema

* Status: accepted (2026-10-13, approved by cerebro with the phase 1 audit)
* Date: 2026-10-13

## Context and problem statement

Mathias must choose what opens each file type, plus columns and theme. JSON was asked for in the order
(`config.json`); it has no comments, so unknown keys are rejected with a clear message and a documented example ships
in `docs/configuration.md`.

## Decision outcome

Location order: `$LAZYFINDER_CONFIG`, `$XDG_CONFIG_HOME/lazyfinder/config.json`, `~/.config/lazyfinder/config.json`.
Missing file = defaults. Invalid file = error naming the field, app starts with defaults. Unknown keys are errors
(typos in `openers` must not silently do nothing). `"version": 1` is required once the file exists, so the format can
evolve (ED-9: the format is public API).

```json
{
  "version": 1,
  "columns": ["name", "modified", "created", "size", "kind"],
  "sort": { "by": "name", "descending": false, "folders_first": true },
  "show_hidden": false,
  "sidebar": {
    "favorites": [
      { "name": "Home", "path": "~" },
      { "name": "Documents", "path": "~/Documents" }
    ],
    "locations": [{ "name": "Macintosh HD", "path": "/" }]
  },
  "openers": {
    "vscode": { "argv": ["code", "{file}"], "mode": "gui" },
    "vim":    { "argv": ["vim", "{file}"],  "mode": "terminal" },
    "excel":  { "argv": ["open", "-a", "Microsoft Excel", "{file}"], "mode": "gui" },
    "preview":{ "argv": ["open", "-a", "Preview", "{file}"], "mode": "gui" }
  },
  "open_with": {
    "txt": "vscode", "md": "vscode", "xlsx": "excel", "png": "preview", "jpg": "preview"
  },
  "preview": { "max_text_bytes": 524288, "images": "auto" },
  "theme": "catppuccin-mocha",
  "language": "auto",
  "screen_background": "theme",
  "mouse": true,
  "keys": {},
  "watch": { "max_dirs": 64, "poll_interval_ms": 2000 }
}
```

* `columns`: any ordered subset of `name, modified, created, size, kind`.
* `open_with` keys are lowercase extensions without the dot; values reference `openers` names; a dangling reference is
  a validation error. Anything not listed opens with the system default (`open`).
* `{file}` is the only placeholder; it must appear as a whole argv element (validated).
* `preview.images`: `auto` (probe), `kitty`, `off`.
* `theme`: a built-in palette name, a user theme name or `terminal` (see ADR 0008); `NO_COLOR` overrides to
  monochrome. *(Amended 2026-10-13: it was `auto`, `light`, `dark`.)*
* `language`: `auto` or `en`, `es`, `pt`, `fr`, `de`, `it`, `ja`, `zh`. `screen_background`: `theme` or
  `terminal`. `mouse`: boolean. `keys`: action to key list. *(Added 2026-10-13, ADR 0008.)*
* `~` and `$HOME` expand in `path` fields only.

### Consequences

* Good: strict validation prevents silent misconfiguration; version field allows migration.
* Bad: JSON has no comments; mitigated by `lazyfinder --print-default-config`.
