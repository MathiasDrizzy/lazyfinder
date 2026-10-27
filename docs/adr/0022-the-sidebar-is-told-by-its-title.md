# How the herdr plugin tells its sidebar from other panes

* Status: superseded by [0023](0023-the-sidebar-is-told-by-its-process.md)
* Date: 2026-10-26

## Context and problem statement

`pane.list` of herdr has no field that says which plugin owns a pane, so the plugin knows its sidebar only by the title
herdr gives it (`lazyfinder`). The herdr.auto-title plugin names panes after their folder, so the pane of the agent
working in the lazyfinder folder carried that title too. The plugin took it for the sidebar: `toggle` with that pane
focused closed it, with the agent inside, and `ensure` never docked a sidebar in that tab.

## Decision

A pane is the sidebar only when it has the title and runs no agent (`Pane.titled`, v0.16.7). An agent pane is never
closed, taken as the anchor or counted as the sidebar. The title and `herdr-plugin.toml` stay as they are.

## Not done, on purpose

- Renaming the title to something unique. Sidebars already open, and the ones herdr restores after a restart, keep the
  old title; the plugin would stop recognizing them and dock a second one in every tab.
- Telling a pane by its process (the root process of the sidebar is lazyfinder). A sidebar herdr restored as a shell is
  indistinguishable from a shell of the user's with that title, and closing the restored one is intended (it would keep
  the tab from getting a sidebar). A shell the user titled `lazyfinder` by hand can still be mistaken for the sidebar.
  The cost is a pane the user named on purpose; a title a plugin gives on its own is covered by the agent check and by
  turning `HERDR_AUTO_TITLE_PANES` off.
