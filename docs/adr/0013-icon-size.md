# Icon size: Nerd Font glyphs and roomy rows, not Kitty file icons

* Status: accepted (2026-10-13, order 2026-10-13-05 revision 2, criterion 3)
* Date: 2026-10-13

## Context and problem statement

The icons look too small next to Finder's. In a terminal a text icon is one cell wide and one cell
tall. Finder draws real, large icons. The order asks to decide, measure and deliver a side-by-side capture, or
explain in an ADR why a real icon is not viable.

## Options

1. **Nerd Font glyphs in one cell** (shipped since 0.8.0, colored by kind, `icons: nerd|ascii|off`).
2. **Roomy rows** (`density: comfortable`): the same glyph with a blank line between rows. Rows are two lines tall,
   so the icon has more air and the list reads like Finder's, but the glyph itself does not grow.
3. **Kitty graphics protocol icons** drawn per row, falling back to option 1.

## Decision outcome

Options 1 and 2 ship; option 3 is **not implemented**.

* Measured here (captures `docs/screenshots/columns-compact.png` and `columns-comfortable.png`, Ghostty, 2026-10-13):
  roomy rows give the list Finder's rhythm; the glyph stays one cell. This does not make the icon bigger. It is
  the closest a text icon gets, and it is a limit to be honest about.
* Why not Kitty file icons (INFERRED, not measured by a prototype): a real macOS file icon has to be fetched from
  the system (`NSWorkspace`, which needs cgo or a helper binary, or Quick Look through `qlmanage`), converted to PNG
  and cached per type; each visible row then needs an image placement that must be moved or deleted on every scroll,
  resize, popup and redraw while Bubble Tea redraws cell by cell. Terminals without the protocol need the glyph
  fallback anyway, and the preview already draws images with the protocol, so the cost is only worth paying if
  users ask for it after seeing the roomy rows.
* Side by side with Finder: Finder is not part of this capture (the tests never drive other applications). The
  lazyfinder half of the comparison is the two captures above.

## Consequences

* The glyph stays small by design. If bigger icons are wanted, the next step is a prototype of option 3 for the
  folder and file-type icons only (a handful of cached PNGs), measured for flicker on scroll; decided by the maintainer.
