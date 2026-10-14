# Icon size is a setting: air, tiles and roomy rows

* Status: accepted (2026-10-13, order 2026-10-13-06, points 2 and 3)
* Date: 2026-10-13
* Amends: [0013](0013-icon-size.md) (which left the size fixed)

## Context and problem statement

After 0.10.0 the request was bigger icons, with a setting to make them bigger or smaller, and the same for the
settings gear, which looked very small. ADR 0013 explained that a text icon cannot grow. The order asks
to explore what is possible in a cell and to leave what was measured.

## Measurement

Ghostty, 2026-10-13, one script printing the folder glyph and the gear with different surroundings:

* A Nerd Font glyph followed by one, two or three blanks is drawn at the same size: blanks add air, not size.
* The Nerd Font gear (`U+F013`) is much bolder and bigger than the text symbol `⚙` (`U+2699`) in the same cell.
* A glyph on a background tile of three cells (blank, glyph, blank) reads as a bigger icon, like an app icon.
* Two glyphs in two cells draw two small glyphs, not one big one.

## Decision outcome

`icon_size` is `small`, `medium` (default) or `large`:

| | list icon | sidebar | gear | rows |
|---|---|---|---|---|
| `small` | glyph and a space (as in 0.10.0) | no icons | `⚙` bare, 4 cells to click | `density` |
| `medium` | glyph and two spaces | icon before each place | Nerd Font gear on a raised tile, 6 cells to click | `density` |
| `large` | glyph on a three-cell tile and a space | icon before each place | gear on a brighter five-cell tile, 8 cells to click | two lines |

Color themes draw the tile with `surface0` (`surface1` for the gear of `large`); `NO_COLOR` and the terminal theme draw
the glyph without tile. With `icons: ascii` the gear stays `⚙`.

* Real file icons through the Kitty protocol stay out, as decided in 0013; this setting is the closest a cell gets and
  it is a limit to be honest about.
* The size changes layout, so the name column is narrower by the extra cells: the default columns still fit at 120
  columns and optional columns are dropped as before.

## Consequences

* A new `icon_size` key; an unknown value is refused by validation with the other keys of the file kept.
* `large` implies two-line rows even when `density` is `compact`.
