# Parity with lazymark

lazymark (same author, same architecture) is the reference for look, texts and behavior. This table lists what
lazyfinder should share with it. Status: **done**, **missing (block X)** = scheduled in order 2026-10-13-05, or
**n/a** with the reason. Texts are checked by `TestStringsShared` against a committed snapshot of lazymark's
strings (`scripts/snapshot-lazymark-strings.py` refreshes it).

| Area | lazymark | lazyfinder | Status |
|---|---|---|---|
| Texts | Settings values `terminal` / `tema` / `auto`, "Terminal muy pequeña" | same wording, all 8 languages, key by key | done |
| Texts | Same i18n architecture (`i18n.T`, JSON catalogs, coverage test) | same | done |
| Icons | Nerd Font one-cell glyphs, folder open/closed, color | glyph per kind (folders open/closed, ~90 extensions, apps, links, hidden dim), color per kind | done |
| Icons | `nerd_font` on/off with text fallback | `icons`: nerd, ascii, off; `icon_map` overrides an extension; Settings row | done |
| Icons | one-cell only | Finder-like taller rows / Kitty icons | missing (block G, row density) |
| Settings popup | rows label left / value right, ←/→/Enter, theme, language, background | same rows plus Icons, Images, Hidden, Folders first, Mouse | done |
| Settings | first click selects, second click changes | a click changes | missing (block H) |
| Settings | "Atajos" Lazy / Lazy + Vim | `keys` remap in config; hjkl always on | n/a: a file manager keeps hjkl and arrows together |
| Settings | "Confirmar al borrar" | trash is undoable (`u`), no confirmation needed | n/a: delete goes to the Trash and can be undone |
| Settings | Mascot, popup background | none | n/a: no mascot; popups follow the theme |
| Sidebar | n/a (notes tree) | places named in the interface language (Inicio, Escritorio, Documentos…), title and breadcrumb too | done |
| Sidebar | n/a | add / remove / reorder / rename favorites, Trash, Recents place | missing (block F) |
| Search | n/a (notes) | field "Buscar" visible top right, scopes, files/folders, name/content | missing (block C) |
| Mouse | divider drag, column `[` `]`, wheel under pointer | wheel, clicks | missing (block E) |
| Mouse | clickable footer key hints | footer is text | missing (block H) |
| Columns | n/a | configurable columns, order, width, date format, header menu | missing (block G) |
| Selection color | blue-gray | peach (theme) | missing (block H) |
| `?` help | popup generated from the keymap, `docs/keybindings.md` generated | `docs/keybindings.md` generated; no popup | missing (block F) |
| Config | per-key tolerant with warnings | one invalid key makes the file fall back | missing (block H) |
| Popups | close only with Esc | click outside closes | missing (block H) |
| Zoom panel `w` | yes | preview is full screen on Space | n/a: Space already zooms |
| Trash popup | restore / delete forever / empty | trash by `⌫`, undo | missing (block F) |
| CLI / MCP | notes CLI and MCP server | `--follow`, `--theme`, `--lang`… | n/a: a notes API makes no sense for files |
| Tabs, themes, user themes, NO_COLOR, `--no-mouse`, completions, man page | yes | yes | done |
