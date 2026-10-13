# File operations: Trash only, never overwrite, always undoable

* Status: accepted (2026-10-13, milestone M4; requirements R-35 and R-51)
* Date: 2026-10-13

## Context and problem statement

Milestone 4 is the first one that writes to the file system. A file manager that loses a file once is not trusted
again, so the design starts from what must never happen: deleting for good, overwriting a file, or leaving a half
done operation that cannot be reversed.

## Decision drivers

* Mathias asked for the same functions as Finder (copy, move, rename, new folder, Trash, duplicate, Get Info).
* The rules of the project: tests run in temporary folders, never on real data; side effects behind a small seam.
* Terminal reality: there is no drag and drop and no modifier-click on every terminal, so the keyboard reaches
  everything and the mouse reaches the common cases.

## Decision outcome

* A new package `internal/fileops` does the work and knows nothing about the UI. The UI keeps one `*fileops.Ops`.
* **No operation deletes for good.** "Delete" is a move into the Trash folder (`ops.trash_dir`, `~/.Trash` by
  default). A name taken in the Trash gets a Finder-style suffix (`a copy.txt`, `a copy 2.txt`). Across file systems
  the item is copied first and the source is removed only after the whole copy succeeded; the user's data is
  always in one of the two places.
* **No operation overwrites.** Copy, move and duplicate pick a free name with the same suffix rule. Rename and new
  folder refuse a name that exists and keep the prompt open so the name can be fixed. A folder cannot be copied or
  moved into itself.
* **Every operation is one undo step**, kept in memory (last 100): copy undoes by trashing what it created, move and
  rename by moving back, new folder by removing it when still empty (else trashing it), Trash by moving back to a free
  name beside the original. A failure half way keeps what was done undoable.
* **Confirmation** is asked only where a mistake is expensive: trashing more than `ops.confirm_trash_above` items at
  once (default 1, so a batch asks and a single file does not, because it is one keystroke from undo).
* **Selection of many** is a set of paths (`v` toggles and moves down, `ctrl+a` all, `esc` clears, ctrl-click toggles
  with the mouse). It is cleared when the folder changes. An operation acts on the marked rows, else on the selected
  row. A row opened in place (R-33) is an ordinary row with its own path.
* Operations on a search result list are refused ("Close the search first") except copy, duplicate, Get Info and
  marking: the results have no folder to refresh.
* The clipboard is internal to the app. Writing file URLs to the macOS pasteboard would touch the user's real
  clipboard, which the project rules forbid in tests, and needs a native dependency.

## Consequences

* Good: nothing real is ever lost; each rule is a unit test on temp folders (`internal/fileops`) and the key flow is
  a tuiqa scenario (`file-ops.json`) in an isolated HOME, whose Trash is the isolated one.
* Cost: the Trash is a plain folder. Finder's "Put Back" metadata is not written, so Finder cannot put an item back
  by itself; lazyfinder's own undo can, during the session. Emptying the Trash is left to Finder or the user.
* Cost: undo history is lost when the program quits. A persistent journal is a candidate for a later milestone.
