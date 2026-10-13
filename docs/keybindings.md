# Keybindings

These are the defaults. Every list action can be rebound with `keys` in `config.json`
(see [configuration](configuration.md#keys)); the action names are in the last column.

## List

| Key | Action | Name |
| --- | --- | --- |
| `↑` `↓`, `k` `j` | Move the selection | `up` `down` |
| `PgUp` `PgDn` | Move a page | `page_up` `page_down` |
| `Home` `g`, `End` `G` | First and last row | `first` `last` |
| `Enter`, `→`, `l` | Enter a folder, or open a file with its configured program | `open` |
| `Backspace`, `←`, `h` | Parent folder | `parent` |
| `Alt+←`, `[` | Back in history | `back` |
| `Alt+→`, `]` | Forward in history | `forward` |
| `s` | Sort by the next column | `sort` |
| `S` | Reverse the sort order | `sort_reverse` |
| `.` | Show or hide dotfiles | `hidden` |
| `I` | Show or hide files ignored by git | `ignored` |
| `/` or `ctrl+f` | Open the search bar (see [search](search.md)) | `search` |
| `+` `=`, `-` | Open or close the selected folder in place; `-` on a row inside an open folder closes that folder. Clicking the `>` marker does the same | `expand` `collapse` |
| `ctrl+g` | Go to a folder: type a path (`~`, relative or absolute), `Tab` completes, `Enter` goes | `goto` |
| `v`, `ctrl+a`, `esc` | Mark the row and move down, mark every row, clear the marks. Ctrl-click marks with the mouse | `mark` `mark_all` |
| `c`, `x`, `p` or `ctrl+v` | Copy, cut, paste into the current folder (marked rows, else the selected one) | `copy` `cut` `paste` |
| `D` | Duplicate in the same folder (`name copy.ext`) | `duplicate` |
| `F2`, `R` | Rename the selected item | `rename` |
| `N`, `ctrl+n` | New folder (the name is asked, `untitled folder` by default) | `new_folder` |
| `d`, `Delete` | Move to the Trash (asks above `ops.confirm_trash_above` items) | `trash` |
| `u`, `ctrl+z` | Undo the last operation | `undo` |
| `i` | Get Info popup | `info` |
| `r` | Reload the folder | `reload` |
| `Tab` | Move focus between the sidebar and the list | `focus` |
| `Space` | Preview popup | `preview` |
| `O` | Open With menu | `open_with` |
| `Q` | Quick Look for the selection | `quick_look` |
| `,` | Settings popup | `settings` |
| `q` | Quit | `quit` |

## Sidebar (after `Tab`)

`↑` `↓` (or `k` `j`) move, `Enter` or `→` opens the place, `Tab` or `Esc` returns to the list.

## Preview popup

`Space` or `Esc` closes it, `↑` `↓` show the previous or next entry, `Enter` opens the file, `Q` starts Quick Look,
`q` quits.

## Settings popup

`↑` `↓` pick a setting, `←` `→` (or `Enter`, `Space`) change it, `Esc` or `,` close it. `Ctrl+C` quits from anywhere.

## Open With menu

`↑` `↓` move, `Enter` runs the chosen program, `Esc` cancels.

## Mouse

| Gesture | Action |
| --- | --- |
| Click on a row | Select it |
| Double click on a row | Enter the folder or open the file |
| Click on a column header | Sort by it; click again to reverse |
| Wheel | Scroll the list (three rows per notch) |
| Click on a sidebar entry | Go to that place |
| Click on a breadcrumb segment | Go to that folder |
| Click on `‹` or `›` | Back or forward |
| Click on the gear `⚙` | Settings popup |
| Click outside a popup | Close it |

`--no-mouse` or `"mouse": false` turns the mouse off.
