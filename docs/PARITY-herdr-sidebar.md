# Parity with the herdr sidebar

What the [herdr sidebar](https://github.com/alexarthurs/herdr-sidebar) offers in its Git view and on its hover
buttons, and what lazyfinder does about each item. Every "done" names its proof in [qa-matriz.md](qa-matriz.md).

## Git tab (sidebar mode)

| herdr sidebar | lazyfinder | State |
|---|---|---|
| Branch name, dirty mark, ahead/behind | header row and bottom bar (`main*`, `↓0 ↑0`, change count) | done |
| Switch branch from a searchable list | branch menu (`B`, click on the branch): search, switch, create, warning that changes go with you | done |
| Create a branch | same menu, typing a new name | done |
| Staged and unstaged changes with counts | `Staged Changes` and `Changes` sections, foldable | done |
| Diff of a file | Enter or the ↗ button | done |
| Stage, unstage, discard | key, row buttons, section buttons; discard asks first, an untracked file is deleted | done |
| Stash | `Stashes` section: stash, pop (Enter), drop with confirmation | done |
| Fetch, pull, push | menu entries and header buttons; pull is fast-forward only; push without upstream explains instead of guessing | done |
| Conflicts | marked `U` and cannot be discarded by mistake | done |
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
