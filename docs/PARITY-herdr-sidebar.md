# Parity with the herdr sidebar

What the [herdr sidebar](https://github.com/alexarthurs/herdr-sidebar) offers in its Git view and on its hover
buttons, and what lazyfinder does about each item. Every "done" names its proof in [qa-matriz.md](qa-matriz.md).

## Look of the explorer (sidebar mode)

Taken from `explorer_app.rs` (`draw`, `draw_activity_bar`, `draw_header`, `row_line`) and `ui.rs` of herdr-sidebar
081ed13.

| herdr sidebar | lazyfinder | State |
|---|---|---|
| No frame of its own: herdr frames and titles the pane | no border and no counter in the sidebar mode | done |
| Activity bar three rows tall, active chip extended with `▄`/`▀` half blocks, hover with the same geometry | same, rows 0–2; every row of a chip takes the click | done |
| Chips `" G  "` (FA glyphs are two cells wide), one space apart, gear at the right edge | same chips, also in the window toolbar, one free cell apart; the mode button sits left of the gear | done |
| Keycap hover `#32363d`, a lighter chip for the active tab; file icons in fixed colors per kind | the theme's surface color for the hover and a step lighter for the active tab; each file kind takes the theme color closest to herdr's, so every part follows the theme | different on purpose (order 07, revision 2) |
| Header: ` NAME` bold in the accent blue, title buttons right-aligned, dimmed, shown on mouse activity for 3 s | same (no refresh button: the list updates by itself) | done |
| Row: indent of two cells a level, dimmed `▸`/`▾`, icon, plain name, git letter or `●` two cells from the edge, ignored names dimmed | same layout; `>`/`v` and the colors of the normal window (Mathias, 2026-10-14: both modes must look like one program) | done, colors different on purpose |
| Blue-grey folder icon `#90a4ae` | the blue folder icon and name of the normal window | different on purpose |
| Selection sets only the background (bold), icons keep their color | same | done |
| Footer: git status and the `«` collapse button | branch and distance from upstream, key hints; no `«` (herdr's own pane control) | partial |

## Git tab (sidebar mode)

| herdr sidebar | lazyfinder | State |
|---|---|---|
| Branch name, dirty mark, ahead/behind | header row and bottom bar (`main*`, `↓0 ↑0`, change count); a detached HEAD reads `HEAD (detached at 1a2b3c4)`; outside a repository no git buttons | done |
| Switch branch from a searchable list | branch menu (`B`, click on the branch): search, switch, create, warning that changes go with you | done |
| Create a branch | same menu, typing a new name | done |
| Staged and unstaged changes with counts | `Staged Changes` and `Changes` sections, foldable | done |
| Diff of a file | Enter or the ↗ button | done |
| Stage, unstage, discard | key, row buttons, section buttons; discard asks first, an untracked file is deleted | done |
| Stash | `Stashes` section: stash, pop (Enter), drop with confirmation | done |
| Fetch, pull, push | menu entries and header buttons; pull is fast-forward only; push without upstream explains instead of guessing | done |
| Conflicts | marked `!` and cannot be discarded by mistake; discard all and stage all name the files in conflict and wait for a yes | done |
| Commit, rebase, merge | not offered on purpose (the order says so); `L` opens lazygit | not done, by design |

## Hover buttons

| Button | Tooltip | Hover | Action | Proof |
|---|---|---|---|---|
| New file, new folder, refresh, collapse (Files tab) | yes | yes | yes | T ActionIconsCreateFilesAndFoldersRefreshAndCollapse, S sidebar-actions |
| Row buttons of the Git tab (open changes, discard, stage / unstage) | yes | yes | yes | T StageUnstageWithTheKeyAndTheButtons, T TheOpenButtonShowsTheDiffAndOShortcutOpensTheFile |
| Section buttons (stage all, unstage all) | yes | yes | yes | T SectionButtonsStageAndUnstageEverything |
| Branch label and sync buttons | yes | yes | yes | T TheBranchMenuSwitchesCreatesAndWarnsAboutChanges, T FetchPullAndPushFromTheTab |
| Tabs, mode button, settings | yes | yes | yes | T ProjectModeHasClickableFilesSearchAndGitTabs, S mode-button |

## What came to the normal (finder) window

| Item | State |
|---|---|
| Branch in the top bar, click opens the branch menu | done |
| Git letters on rows | done |
| Right-click on a changed file: open with, view changes, stage, unstage, discard (with confirmation) | done (T RightClickOnAChangedFileOffersTheGitActions) |
| Staged/unstaged/stash sections | not done: they need a tree of their own, which the columns do not have; use the sidebar mode (`ctrl+b`) |
