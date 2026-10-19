# Improvements of the 2026 audit

* Status: accepted
* Date: 2026-10-19

## Context and problem statement

The full audit (branch `audit/full-sweep`) compared lazyfinder with yazi, ranger, nnn, lf and the Finder and listed
candidate improvements. Each one ships with tests and docs, so none stays open as "later".

## Shipped

* **Long copies run apart from the UI.** Paste and Duplicate show the bytes copied of the total in the status line, and
  `esc` cancels without leaving a half copy. Any other change to the files waits until the copy ends, and quitting
  stops it first. Before, a copy of gigabytes froze the program.
* **Get Info counts a folder apart** ("Counting…") and shows the owner.
* **Copy the path** of the selection or of the marked rows with `y` (`copy_path`), like `yy` in ranger and lf.
* **`high-contrast` theme**: every text color at 7:1 at least (WCAG AAA) on the background and on both selections.
* **Linux**: the XDG Trash with its `.trashinfo`, Linux places in the sidebar, and `xdg-open` as the default opener.
* Already there before the audit, and kept: marking many rows (`v`, `ctrl+a`) for every operation, bookmarks as
  Favorites (`b`), hex dumps of binaries, previews of archives, Markdown and highlighted JSON, YAML and TOML, PDF pages
  through an external rasterizer, a question before a Trash above `ops.confirm_trash_above` items, and before git
  discards.

## First turned down, then shipped

A first draft turned the six items below down; the owner asked for all of them, so they ship with tests and docs.

* **A fuzzy command palette** (`:` or `ctrl+p`, `palette`). Every action with its keys, worded in the user's language;
  typing narrows the list by a fuzzy match on the name or on the action's name in `keys`. The cursor moves are left
  out, since a key does them better.
* **Vim-style marks** (`m` and a letter or a digit, `'`). The draft's objection was a hidden state, so `'` lists the
  marks before going to one: the mark's letter or `Enter` goes, `delete` removes it. Kept in `marks.json`.
* **Redo and a visible history** (`U` or `ctrl+r`, `H`). Each step that can be undone keeps how to do it again, through
  the same operation that made it, so a redo meets the same checks (no overwrite, a free name). A new operation drops
  what could be redone; a redo that fails before doing anything stays to redo.
* **chmod from the UI** (`P`). Three octal digits, starting from the current mode; it keeps setuid, setgid and sticky,
  never acts through a link, and is undoable.
* **CSV and TSV as a table** in the preview. It reads only what the preview already reads (the byte budget), drops a
  last row cut by that budget, caps a cell at 40 columns and falls back to text when the file is not valid CSV.
* **A drawn progress bar** for a long copy, in the status line next to the bytes copied, so it takes no row.

## Consequences

The improvements above are in the CHANGELOG. Nothing from the audit's list is left out.
