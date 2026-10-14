# No icon size: a glyph in a cell cannot grow

* Status: accepted (2026-10-13, order 2026-10-13-07, point B)
* Date: 2026-10-13
* Supersedes: [0014](0014-icon-size-setting.md); confirms [0013](0013-icon-size.md)

## Context and problem statement

0.11.0 shipped `icon_size` (`small`, `medium`, `large`). Trying it showed two problems: the icons grow
badly, and they leave a square around the icon. Both are real: a terminal cell holds one glyph of a fixed
size, so "medium" and "large" only added blank cells and blank lines around the same glyph, and the tile painted a box
behind it, which lazymark never does.

## Decision

Remove the setting. An icon is a glyph and a space on the plain background, the gear is a bare glyph in the accent
color with a four-cell click zone, and the rows are one line (two in the `comfortable` density, which is its own
setting). The sidebar shows its icons at all times.

## Options measured

* Blank cells and lines around the glyph (0.11.0): looks like a bug, not like a bigger icon. Rejected.
* A tile (a background behind the glyph): the box that was rejected. Rejected.
* A real image per icon with the Kitty graphics protocol, as lazymark does for pictures: needs a font rasterizer or a
  set of bitmaps per theme and per terminal cell size, and falls back to the same glyph anywhere else. Not worth a
  second icon system for the sizes of a glyph; it can come back as its own decision if a bitmap icon set is wanted.

## Consequences

* `icon_size` stays in the file format as an ignored key so a file written by 0.10 or 0.11 still loads.
* The Settings row, the tiles and the framed gear are gone.
