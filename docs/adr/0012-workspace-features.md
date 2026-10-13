# Workspace features: tabs, bookmarks, tags, saved searches, column view

* Status: accepted (2026-10-13, milestone M6; requirement R-53)
* Date: 2026-10-13

## Context and problem statement

M6 adds the conveniences of a daily file manager. Each must keep the rules of the project: nothing real is
touched by tests, nothing runs through a shell, and a failed side file never stops navigation.

## Decision outcome

* **Tabs** keep only the navigation state (history, sort, hidden files, open folders, selected name). Marks, the
  clipboard, undo and popups belong to the window, so a copy in one tab can be pasted in another.
* **Bookmarks** are the sidebar Favorites. `b` edits `sidebar.favorites` in `config.json` through the same atomic
  writer as the settings popup and writes paths under the home folder as `~/…`.
* **Recents, saved searches and tags** are small JSON files next to `config.json` (`recents.json`, `searches.json`,
  `tags.json`). Without a config path (tests) nothing is written. A damaged file starts the list again.
* **Tags are lazyfinder's own**, not Finder's. Finder's tags are a binary property list in an extended attribute;
  writing it needs a plist encoder and breaks the rule of never touching the user's files outside an explicit
  operation. Cost: tags are not visible in Finder and follow a rename done in lazyfinder only; a move, a trash or a
  rename elsewhere leaves the old path behind in `tags.json`.
* **Custom commands** reuse the opener rules: fixed argv, `{file}` and `{dir}` whole elements, a mode, no shell.
* **Quick filter** is a name filter applied when a folder is read; it ends when the folder changes.
* **Column view** (`layout: columns`, key `M`) draws the parent folder, the folder and a preview of the selection
  (the names inside a folder, or the first 8 KiB of a text file) in the same panel. The side columns are refreshed
  after each update, not while drawing. The detail columns, tree expansion and column sorting by click are the list
  layout's; the keyboard works the same in both.
* **ripgrep** speeds up the `in:` content filter (40 ms against 250 to 660 ms on 11 000 files, measured here) and the
  built-in scan is the fallback, so nothing depends on it. **fd is not used**: the built-in walk took 270 ms on the
  same tree against 70 ms, which is fine below about a hundred thousand files, and fd's `.gitignore` rules would make
  a name search show different files than the folder does.

## Consequences

* Good: every feature is a unit test on temporary folders and one tuiqa scenario (`workspace.json`).
* Cost: tags can go stale after outside changes; a cleanup command is a candidate for later.
* Cost: the column view has no column header sorting.
