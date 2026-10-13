# Package layout and data flow

* Status: accepted (2026-10-13, approved by cerebro with the phase 1 audit)
* Date: 2026-10-13

## Context and problem statement

The file model, the view, the watcher and the side effects (opening files, spawning Quick Look) must be separable so
each can be tested alone and the watcher can be added in M2 without touching the view.

## Decision outcome

Chosen layout (Go module `github.com/MathiasDrizzy/lazyfinder`):

```
cmd/lazyfinder/       main: flags, wiring, exit codes
internal/fsmodel/     Entry, Listing, sort comparators, Kind table, human sizes/dates (pure, no UI)
internal/history/     back/forward stack (pure)
internal/config/      load, validate, defaults for config.json (ADR 0004)
internal/opener/      resolve opener by extension, run argv without a shell (ADR 0005)
internal/preview/     text/dir/image loaders, async and cancelable (ADR 0006)
internal/termimg/     terminal capability probe and Kitty graphics output
internal/watch/       Watcher interface + no-op implementation in M1; fsnotify/git in M2 (ADR 0003)
internal/ui/          Bubble Tea model, views, layout, mouse hit-testing; the only package that imports charm.land
```

Rules:

* **Model never imports UI.** `fsmodel`, `history`, `config`, `opener` have no terminal dependency and are tested
  with plain `go test`.
* **Side effects behind interfaces** (`opener.Runner`, `preview.Loader`, `watch.Watcher`, `fs.FS`-style reader) so tests
  inject stubs and fixtures; no test touches the real home (R-23).
* **Unidirectional flow:** UI emits intents (`Navigate`, `Open`, `Sort`) and receives messages (`ListingLoaded`,
  `PreviewReady`, `FsChanged`). A listing is an immutable value; a change produces a new `Listing` (coding style:
  immutability).
* **Mouse:** `ui` keeps a per-frame hit map (rectangle to action), built while rendering, so click handling is a
  lookup and is unit-testable without a terminal.
* Files stay under 400 lines typical, 800 hard.

### Consequences

* Good: M2 adds `watch` and a status column without changing `fsmodel`'s API beyond an optional `GitStatus` field.
* Bad: more packages than a 1 000-line app needs; justified by the roadmap (live follow, operations).
