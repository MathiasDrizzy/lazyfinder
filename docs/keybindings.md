# Keybindings

These are the defaults. Every list action can be rebound with `keys` in `config.json`
(see [configuration](configuration.md#keys)); the action names are in the last column.

## List

| Key | Action | Name |
| --- | --- | --- |
| `↑` `↓`, `k` `j` | Move the selection | `up` `down` |
| `?` `F1` | Help with every key and gesture | `help` |
| `b` | Add the folder to Favorites, or remove it (also from the sidebar) | `bookmark` |
| sidebar: `d`, `F2`, `K` `J` | Remove, rename, move up, move down the favorite | |
| `<` `>` | Narrow / widen the sidebar | `sidebar_narrow` `sidebar_widen` |
| `ctrl+up` `ctrl+down` | Shorter / taller Favorites box of the sidebar (or drag its border) | `favorites_shorter` `favorites_taller` |
| `ctrl+left` `ctrl+right` | Narrow / widen the column the list is sorted by | `column_narrow` `column_widen` |
| `PgUp` `ctrl+u`, `PgDn` `ctrl+d` | Move a page | `page_up` `page_down` |
| `Home` `g`, `End` `G` | First and last row | `first` `last` |
| `Enter`, `→`, `l` | Enter a folder, or open a file with its configured program | `open` |
| `Backspace`, `←`, `h` | Parent folder | `parent` |
| `Alt+←`, `[` | Back in history | `back` |
| `Alt+→`, `]` | Forward in history | `forward` |
| `s` | Sort by the next column | `sort` |
| `S` | Reverse the sort order | `sort_reverse` |
| `.` | Show or hide dotfiles (the sidebar tree never lists `.git`, as herdr) | `hidden` |
| `I` | Show or hide files ignored by git | `ignored` |
| `/` or `ctrl+f` | Open the search bar (see [search](search.md)) | `search` |
| `+` `=`, `-` | Open or close the selected folder in place; `-` on a row inside an open folder closes that folder. Clicking the `>` marker does the same | `expand` `collapse` |
| `ctrl+g` | Go to a folder: type a path (`~`, relative or absolute), `Tab` completes, `Enter` goes | `goto` |
| `v`, `ctrl+a`, `esc` | Mark the row and move down, mark every row, clear the marks. Ctrl-click marks with the mouse | `mark` `mark_all` |
| `c`, `x`, `p` or `ctrl+v` | Copy, cut, paste into the current folder (marked rows, else the selected one). A paste or a duplicate runs apart with a progress bar in the status line; `esc` cancels it and leaves no half copy, and quitting stops it first | `copy` `cut` `paste` |
| `y` | Copy the path of the selected row, or of the marked rows one per line, to the clipboard | `copy_path` |
| `P` | Change the permissions of the selected or marked items, typed in octal (`644`, `755`); not those of what a folder holds, never on a link; `u` undoes it | `chmod` |
| `D` | Duplicate in the same folder (`name copy.ext`) | `duplicate` |
| `F2`, `R` | Rename the selected item | `rename` |
| `N`, `ctrl+n` | New folder (the name is asked, `untitled folder` by default) | `new_folder` |
| `n` | New empty file (the name is asked, `untitled file` by default) | `new_file` |
| `d`, `Delete` | Move to the Trash (asks above `ops.confirm_trash_above` items) | `trash` |
| `u`, `ctrl+z` | Undo the last operation | `undo` |
| `U`, `ctrl+r` | Redo what the last undo reversed; a new operation drops what could be redone | `redo` |
| `H` | History: what `u` can undo, the next one first, and then what `U` can redo | `history` |
| `i` | Get Info popup | `info` |
| `r` | Reload the folder | `reload` |
| `ctrl+t`, `ctrl+w` | New tab (a copy of this one), close the tab; the last tab stays | `new_tab` `close_tab` |
| `}`, `{` | Next and previous tab (wraps around) | `next_tab` `prev_tab` |
| `b` | Bookmark: add the selected folder (or this one) to Favorites, or remove it. Saved in `config.json` | `bookmark` |
| `E` | Recent files and folders (the last 20 you went to or opened, kept in `recents.json` next to the config) | `recents` |
| `m` then a letter, `'` | `m` and a letter or a digit mark the current folder, like Vim's marks; `'` lists the marks, and the mark's letter (or `Enter`) goes to its folder, `delete` removes it. Kept in `marks.json` next to the config | `set_mark` `jump_mark` |
| `f` | Quick filter by name or glob (`*.go`); `Enter` keeps it, `Esc` drops it. Ends when you leave the folder | `filter` |
| `F` | Saved searches; `Delete` removes one. Save the search being typed with `ctrl+s` | `saved_searches` |
| `t`, `T` | Tag the marked rows (again to untag), list the tags in use and search for one | `tag` `tags` |
| `!` | Custom commands from `commands` in `config.json` | `commands` |
| `M` | Switch between the detail list and the column view (parent, folder, preview) | `layout` |
| `Tab` | Move focus between the sidebar and the list | `focus` |
| `Space` | Preview popup | `preview` |
| `O` | Open With menu | `open_with` |
| `Q` | Quick Look for the selection (on Linux, the default app through `xdg-open`) | `quick_look` |
| `Ctrl+B` | Project mode: the folder as a tree with git state, like the VS Code explorer (again to leave) | `project` |
| `B` | Branch menu of the normal window: type to search, `Enter` switches, "Create branch" makes a new one; changes you have not saved are named before the switch. The branch in the toolbar opens it too. The sidebar mode has none | `branch` |
| `,` | Settings popup | `settings` |
| `q` | Quit | `quit` |

Moving past the last row goes to the first, and past the first goes to the last (in the list, the tree, every menu and
the sidebar). Right click on a file with git changes adds Show changes to the menu, and in the normal window also Stage,
Unstage and Discard changes.

## Sidebar mode: the Git tab

The tab is a quick look at what changed: **Staged Changes**, **Changes** (conflicts are marked `!`) and **Stashes**
(folded until opened), each with its count. The selected row, and the row under the pointer, show the `↗` button in
place of the git letter. The tab changes nothing: staging, discarding, stashing, switching branches, pulling, pushing
and committing are left to lazygit. These keys work on the tab and are not rebindable:

| Key | Action |
| --- | --- |
| `↑` `↓` | Move (wraps around) |
| `←` | On a section heading, fold it |
| `→`, `Enter`, `Space` | On a heading, fold or unfold it; on a file, show its diff |
| `↗` | Show the diff of the file; `Esc` closes it |
| `o` | Open the file with its program |
| `L` | Open lazygit, when it is installed |

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
| Right click on a breadcrumb segment | Copy the path of that folder (pbcopy, wl-copy, xclip or xsel; OSC 52 without them) |
| Click on `‹` or `›` | Back or forward |
| Click on the gear `⚙` | Settings popup |
| Click outside a popup | Close it |

`--no-mouse` or `"mouse": false` turns the mouse off.
