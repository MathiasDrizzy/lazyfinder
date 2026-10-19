# Improvements of the 2026 audit: what ships, what does not

* Status: accepted
* Date: 2026-10-19

## Context and problem statement

The full audit (branch `audit/full-sweep`) compared lazyfinder with yazi, ranger, nnn, lf and the Finder and listed
candidate improvements. Each one is either shipped with tests and docs or turned down here with its reason, so none
stays open as "later".

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

## Turned down

* **A fuzzy command palette.** The keys are listed in the help (`?`), the custom commands have their own menu (`!`),
  and the search bar already finds files by name. A palette would add a second way to reach the same actions and a
  fuzzy matcher to maintain. It is not worth that.
* **Vim marks (`m` + letter, `'` + letter).** Favorites (`b`, with click or `Enter` in the sidebar) and Recents (`E`)
  cover going back to a folder. Marks would add a hidden, per-letter state that is not shown anywhere, unlike the
  Finder's model the project follows (ADR 0009).
* **Redo and a visible undo history.** Every undo moves items back through the Trash or an exclusive rename (ADR
  0010). Redoing an undo would need the reverse of each of those steps, with its own chance of finding the target
  taken, for an action the Finder offers only one level deep. The single undo stack stays, and an undo that fails
  halfway resumes where it stopped.
* **chmod from the UI.** Get Info shows the permissions and the owner. Changing them is rare in a Finder-like browser,
  easy to get wrong on a folder tree, and a terminal is next to the program for `chmod`. Not added.
* **CSV laid out as a table in the preview.** The text preview shows a CSV as it is, which is what an editor shows.
  Aligning columns means reading the whole file to measure them, against the preview's byte budget. Not added.
* **A drawn progress bar.** The status line shows the bytes copied of the total. A bar would take a row out of the
  list for an operation that usually ends in under a second.

## Consequences

The improvements above are in the CHANGELOG. A future request for one of the turned-down items starts from its reason
here.
