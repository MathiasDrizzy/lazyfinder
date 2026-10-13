# Look, themes, languages and settings

* Status: accepted (2026-10-13, order from Mathias relayed by cerebro: "la estetica tiene que ser parecido a finder
  de mac, pero con la estetica lazy como la tenemos en lazymark, ademas tiene que ser todo configurable ... ya la
  rueda la inventamos con lazymark")
* Date: 2026-10-13

## Context and problem statement

lazyfinder must look like Finder in content and like lazymark in style, and everything must be configurable:
themes, languages, keys, mouse, the image protocol. lazymark already has the systems; reinventing them wastes work
and makes the two apps drift.

## Considered options

* Import lazymark as a Go module. Rejected: its packages are `internal/`, it is an application, and the coupling
  would force lock-step releases.
* Copy the small, self-contained pieces with their tests and adapt them. Chosen.
* Write new ones. Rejected by the order.

## Decision outcome

* **Themes**: copy lazymark's `Palette` (14 colors), the 14 built-in palettes and the user-theme loader (one
  `<name>.json` per theme in `<config dir>/themes`, name `[a-z0-9._-]+`, regular files only, 64 KiB cap, problems
  become warnings, theme names are sanitized before they reach the terminal). The package `internal/theme` has no
  global styles: `internal/ui` builds its styles from the active palette. `terminal` means no palette: ANSI
  colors only. `NO_COLOR` forces monochrome whatever the theme says.
* **Languages**: copy lazymark's `i18n` (`T(es, en)`, `Detect`, `Parse`, JSON catalogs keyed by the English text).
  Interface texts are written as pairs, es and en come from the code, the other six from `catalog/*.json`. A
  coverage test parses the sources and fails when a text has no translation in a catalog.
* **Settings**: a popup edits the user-facing options and writes `config.json` atomically (temp file in the same
  folder, then rename), keeping every other key as it was. A run-only override (`--theme`, `--lang`, `--no-mouse`)
  is never saved.
* **Keys**: actions have default keys; `keys` in the config replaces the keys of an action. Conflicts are reported
  and the default stays. The bar of keys and the help are generated from the effective map.
* **Kitty**: the module is already adapted from lazymark's approach (ADR 0006); the choice `auto`, `kitty` or `off`
  is a setting.
* **Look**: two rounded panels (sidebar, list), the focused one with the accent border, the title in the top
  border, `n of m` in the bottom border, a key bar at the bottom. The selected row is a solid bar (accent
  background); without color it is reverse video with the pointer (R-21).

### Consequences

* Good: same look and behavior as lazymark where it matters, tested pieces, a configurable app.
* Bad: the copied code can drift from lazymark; mitigated by keeping it small and by its own tests.
