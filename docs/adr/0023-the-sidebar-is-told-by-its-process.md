# The herdr plugin tells its sidebar by the program it runs

* Status: accepted
* Date: 2026-10-27
* Supersedes: [0022](0022-the-sidebar-is-told-by-its-title.md)

## Context and problem statement

`pane.list` of herdr has no field that names the plugin that owns a pane, so the plugin used the title of the pane
(`lazyfinder`). A plugin that names panes after their folder gave that title to the pane of the agent working in the
lazyfinder folder (the toggle closed it, three times) and renamed a real sidebar to `lazymark › lazyfinder` (the tab
docked a second one). 0022 patched the first case by excluding agent panes and left the rest, because the title was
thought to be the only evidence of a sidebar that herdr restored as a shell.

## Decision

A pane is the sidebar when the root process of the pane (`shell_pid` of `pane.process_info`) is lazyfinder, whatever
the title (`Client.Panes` sets `Pane.Sidebar`). The name comes from the foreground list when the root is in front, else
from `ps -o comm=` (an editor or another program in front of the root does not hide it). Panes that run an agent are
not probed.

* The toggle, `ensure`, the anchor and the tab that closes with its program only act on `Pane.Sidebar`.
* The one pane closed on a guess is a sidebar herdr restored: an idle shell (the root process in front is not
  lazyfinder) that still has the title `lazyfinder` and runs no agent (`Pane.Stale`). A pane with a program in front, a
  pane that cannot be read and a pane with another title are never closed.

## Not done, on purpose

* A title that cannot be taken by another pane. Sidebars already open and the ones herdr restores keep the old title.
* Closing nothing that is not proven to be ours. A restored sidebar is indistinguishable from an idle shell the user
  titled `lazyfinder` by hand, and leaving the dead shell in every tab after each herdr restart is worse than that
  cost.

## Consequences

* A renamed sidebar is still the sidebar, and no second one is docked.
* The plugin makes one `pane.process_info` call per pane that runs no agent each time a hook runs (a handful of
  panes), and `ps` only when the root is not in front.
