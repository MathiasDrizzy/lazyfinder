# Parity with lazymark

lazymark (same author, same architecture) is the reference for look, texts and behavior. One row per thing it has.
Status: **done**; **missing (block X)** = scheduled in order 2026-10-13-05; **n/a** with the reason. Texts are checked
by `TestStringsShared` against a committed snapshot of lazymark's strings (`scripts/snapshot-lazymark-strings.py`
refreshes it). Source of each section: lazymark `docs/keybindings.md`, `docs/configuration.md`, `README.md`, the
Settings popup, the popups and `internal/app/mouse.go`.


## Keys: global

| lazymark | lazyfinder | Status |
|---|---|---|
| `q` `Ctrl+C` Quit | `q`, `ctrl+c` | done |
| `?` Keybindings popup, generated from the keymap | no popup; `docs/keybindings.md` is generated | missing (block F) |
| `Ctrl+V` Paste image | `ctrl+v` pastes the copied files | n/a: a file manager pastes files, not images into a note |
| `,` Settings | `,` | done |
| `x` Trash popup | `x` is Cut (Finder-like); Trash popup with restore/empty | missing (block F, Trash place) |
| `/` Search | `/`, `ctrl+f` | done |
| `T` Daily note | - | n/a: no notes |
| `1` `2` `3` `4` panel jump | `tab` moves sidebar ↔ list | n/a: two panels |
| `Tab` / `Shift+Tab` next / previous panel | `tab` (two panels, so one key is enough) | done |
| `w` Zoom panel | `space` previews full screen, `M` column view | n/a: Space already zooms |
| `W` Kanban board | - | n/a: no tasks |
| `[` `]` Narrow / widen column | `[` `]` are Back / Forward in history | done (block E: `<` `>` resize the sidebar, `ctrl+left` `ctrl+right` the sorted column; the history keys keep their Finder meaning) |
| `Esc` Clear filter | `esc` ends the filter and the search | done |

## Keys: navigation

| lazymark | lazyfinder | Status |
|---|---|---|
| `↑` `k`, `↓` `j` | same | done |
| `←` `h`, `→` `l` | same (parent / open) | done |
| `Home` `g`, `End` `G` | same | done |
| `PgUp` `Ctrl+U`, `PgDn` `Ctrl+D` | same (ctrl+u / ctrl+d added in this audit, `TestVimPagingKeysArePartOfTheDefaults`) | done |
| Vim keys can be turned off in Settings ("Atajos": Lazy / Lazy + Vim) | keys are remapped from `keys` in config.json; no Settings row | missing (block F: Keybindings row with the same two values) |

## Keys: panels that exist only in lazymark

| lazymark | lazyfinder | Status |
|---|---|---|
| Notes panel: `Enter` open, `e` edit, `c` new note, `C` from template, `F` new folder, `r` rename, `m` move, `d` delete, `v` select, `V` select all | `enter` open, `N` new folder, `F2` rename, `d` trash, `v` mark, `ctrl+a` mark all, `c`/`x`/`p` copy, cut, paste (move) | done (file-manager equivalents) |
| Notes panel: new note, from template | - | n/a: not a notes app |
| Tasks panel (`Space`, `H`, `d`, `f`, `a`, `A`) | - | n/a: no tasks |
| Categories panel (`Enter` filters notes) | `T` lists tags and searches one | done (tags exist as Finder tags) |
| Preview panel: `Enter` `e` edit, `n` `N` next / previous link | `space` preview, `O` Open With | n/a: links and editing are note features |
| Kanban board keys (`H` `L` `K` `J`, `o`, ...) | - | n/a: no board |

## Keys: popups

| lazymark | lazyfinder | Status |
|---|---|---|
| Lists inside popups: `Enter` accept | `enter` in every popup | done |
| Trash popup: `r` restore, `d` delete forever, `D` empty trash | `u` undoes a trash; no popup | missing (block F) |
| Confirmation popups: `y` `Enter` confirm | `y` `enter` | done |
| Popups close only with `Esc`; a click outside does nothing | a click outside closes the popup | missing (block H: same as lazymark) |
| Move popup / folder picker (`..`, `s`) | `ctrl+g` goto with path completion; `O` Open With | done (different tool, same job) |
| Dates popup | - | n/a: no tasks |

## Mouse

| lazymark | lazyfinder | Status |
|---|---|---|
| Click a row, tag, popup row or footer hint selects or runs it | rows, sidebar, crumbs, header, chips; footer hints are text | missing (block H: clickable key hints) |
| Drag the divider between the two columns | not draggable | done (block E: sidebar border, preview border of the column view, each column header) |
| Double click the divider | - | done (block E: fit to content, an addition over lazymark) |
| Scroll wheel scrolls the panel under the pointer | wheel scrolls the list or the preview | done |
| Settings: first click selects, second click changes | one click changes | missing (block H) |
| Click on the mascot | - | n/a: no mascot |
| Right click | right click opens the Open With menu on a row | done (lazymark has no context menu); header and sidebar menus in blocks F and G |
| `--no-mouse` / `mouse_click` | `--no-mouse` / `mouse` | done |

## Settings popup rows

| lazymark | lazyfinder | Status |
|---|---|---|
| Tema / Theme | Tema | done |
| Idioma / Language ("auto · Name") | Idioma (values `auto` and the language name) | done |
| Editor | `commands` and Open With programs in config.json | n/a: a file manager opens files with Open With |
| Atajos / Keybindings | - | missing (block F) |
| Alcance de tareas / Task scope | - | n/a: no tasks |
| Confirmar al borrar / Confirm deletion | `ops.confirm_trash_above` in config.json (the Trash is undoable) | missing (block H: Settings row "Confirmar al borrar") |
| Fondo de pantalla / Screen background (`terminal` / `tema`) | same row and values | done |
| Fondo de popups / Popup background | popups follow the theme | missing (block H) |
| Mascota, Tarjetas, fechas, colores de fechas, avisos, glifos, carpeta diaria, plantillas, etiqueta del tablero | - | n/a: notes and tasks only |
| Carpeta de notas / Notes folder | start folder is the argument or the last one | n/a: a file manager has no notes folder |
| Días en la papelera / Days in trash | Trash is the macOS Trash | n/a: the system decides |
| Orden de las notas / Orden de las tareas | Sort by any column (`s`, header click) | done |
| yes / no values ("sí" / "no") | "sí"/"yes" and "no" (fixed in this audit; they said on/off) | done |
| `←` `→` change, `Enter` change, `Esc` close; hint line "←/→ cambiar · clic: seleccionar, 2.º clic: cambiar" | `←` `→` `enter` `esc`; no hint line | missing (block H) |
| - | Icons, Images (Kitty), Hidden files, Folders first, Mouse rows | done (file-manager settings) |

## config.json keys

| lazymark | lazyfinder | Status |
|---|---|---|
| `theme`, `language`, `screen_background`, `mouse_click` | `theme`, `language`, `screen_background`, `mouse` | done (`mouse_click` is named `mouse`) |
| `popup_background` | - | missing (block H) |
| `confirm_delete` | `ops.confirm_trash_above` | missing (block H: a boolean row, same name semantics) |
| `keybinding_mode` (`dual`, `lazy`) | - | missing (block F) |
| `keybindings` (one key per action) | `keys` (a list of keys per action, conflicts reported) | done |
| `sidebar_ratio` 0.15 to 0.75 | fixed 22 columns | done (block E: `sidebar_width`, `preview_width`, `column_widths`, saved on drag) |
| `nerd_font` | `icons` nerd / ascii / off, `icon_map` | done |
| Per-key tolerant parsing: a wrong value goes back to its default, a warning says which, the rest is kept; unknown keys are preserved on save; an invalid file is saved as `config.json.bak` first | one invalid key makes the whole file fall back to the defaults; unknown keys are an error | missing (block H) |
| User themes in `themes/` next to config.json | same | done |
| `notes_dir`, `editor`, `kanban_*`, `date_*`, `due_soon_days`, `done_tag*`, `daily_*`, `templates_folder`, `trash_days`, `max_note_mb`, `mascot`, `click_hint*`, `task_scope`, `hide_completed_*`, `show_tasks_tab`, `show_tags_tab`, `notes_sort`, `tasks_sort` | - | n/a: notes and tasks |
| Config path per OS (`~/Library/Application Support/lazymark/config.json` on macOS) | `~/.config/lazyfinder/config.json` or `LAZYFINDER_CONFIG` | done (documented; same XDG idea) |

## Command line

| lazymark | lazyfinder | Status |
|---|---|---|
| `--dir <folder>` | `lazyfinder [folder]` | done |
| `--theme <name>` | `--theme` | done |
| `--no-mouse` | `--no-mouse` | done |
| `--version` | `--version` | done |
| `note`, `search`, `task`, `paste`, `editor-plugins`, `mcp`, `daily`, `dates`, `kanban` commands | `--follow`, `--lang`, `--print-default-config`, `--completions` | n/a: those commands expose notes and tasks to scripts and agents |

## Look and texts

| lazymark | lazyfinder | Status |
|---|---|---|
| Selection bar blue-gray (theme `surface1`) | selection bar peach | missing (block H) |
| Border of the focused panel green, others dim | same | done |
| Footer key hints (clickable) | key hints on the bottom row, not clickable | missing (block H) |
| Nerd Font icons per kind with color | same (block B) | done |
| Screen size guard "Terminal muy pequeña" | same text | done |
| Texts: same i18n architecture, 8 languages, coverage test | same, plus `TestStringsShared` | done |
| Text of every shared string identical (Settings values `terminal` / `tema` / `auto`) | identical (fixed in this audit) | done |
| NO_COLOR honored | honored | done |
| Contrast sweep over every theme | `TestTextReadsInEveryTheme` | done |
| Screen background `theme` paints every cell | same | done |

## Search, places and columns (not in lazymark, required by the order)

| lazymark | lazyfinder | Status |
|---|---|---|
| Search `/` over the notes with `lazymark search` | search bar at the bottom, `/`; visible Search field at the top right; scope (folder, subfolders, system via Spotlight), files/folders, name/content, saved in config.json and in Settings | done (block C) |
| - | Favorites: `b` bookmarks the current folder | done; right-click menu, remove, reorder, rename: block F |
| - | Columns: fixed list, sort by click | missing (block G: choose, order, width, date format, header menu) |
| - | Places named in the interface language | done |
| - | Recents place in the sidebar | done (block C) |
