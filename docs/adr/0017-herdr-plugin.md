# lazyfinder as a herdr plugin

* Status: accepted
* Date: 2026-10-15

## Context and problem statement

The maintainer works in [herdr](https://herdr.dev) and asked for lazyfinder to behave like the herdr-sidebar plugin:
a sidebar opened by itself in every tab, on the left or the right as set in the settings, following the folder of the
terminal next to it, and in the sidebar mode. Run from a terminal, lazyfinder must keep both modes.

## Decision

* `herdr-plugin.toml` sits at the root of the repository, so `herdr plugin install MathiasDrizzy/lazyfinder` works. Its
  build step (`scripts/herdr-build.sh`) builds `bin/lazyfinder` with Go, or links an installed lazyfinder that knows
  `--herdr`.
* The hooks of `tab.created`, `workspace.created`, `tab.focused`, `workspace.focused` and `pane.focused` run
  `lazyfinder --herdr ensure` (package `internal/herdr`). It takes a non-blocking `flock` in the plugin's state folder
  (the layout changes of a dock fire more hooks, which then return at once), finds the tab of the event, and docks a
  sidebar when the tab has none, the user did not close it there, and `herdr.auto_open` is on. The same flow as
  herdr-sidebar: `plugin.pane.open` (it only splits right), `pane.swap` for a left dock, `pane.resize` to
  `herdr.width`, and `pane.focus` back to the pane that had the focus.
* The pane of the plugin is told apart by its label, the `title` of the `[[panes]]` entry: `pane.list` has no owner
  field (measured on herdr 0.9.4).
* The sidebar follows the folder by asking `pane.list` every 500 ms for the `foreground_cwd` (else `cwd`) of the
  focused pane of its tab, and goes there only when that folder changes, so a folder opened by hand stays. It only
  does this when `HERDR_PLUGIN_ID` is `lazyfinder`; a lazyfinder started by hand inside herdr behaves as anywhere.
* Settings has "Sidebar in herdr" (left or right); `herdr.auto_open` and `herdr.width` live in `config.json`.

## Consequences

* A tab whose dock edge is already split in two rows gets the sidebar next to the top pane only; herdr-sidebar
  repairs this by moving panes through a temporary tab, which is left out until someone needs it.
* A sidebar keeps its edge when the setting changes; new tabs use the new one.
* The poll costs one socket call every 500 ms per sidebar while it runs.
* Verified on 2026-10-15 in an isolated herdr 0.9.4 (own HOME and socket): a new tab got one sidebar on the left at
  25 % with the focus left on the terminal, it followed a `cd`, the toggle closed it and the hooks left it closed,
  the toggle opened it again, and with `herdr.dock` set to `right` a new tab got it on the right.
