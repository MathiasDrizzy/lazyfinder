# Preview pipeline and image protocol

* Status: accepted (2026-10-13)
* Date: 2026-10-13

## Context and problem statement

Space must show a Quick Look-like popup for text, images and folders without blocking the UI, in Ghostty (Kitty
graphics) and degrade elsewhere. The author's session runs inside a multiplexer (`TERM_PROGRAM=herdr`), so environment
variables alone cannot identify the terminal.

## Decision outcome

* **Pipeline:** selection change starts `preview.Load(ctx, entry)` in a goroutine; moving the selection cancels the
  context; results arrive as `PreviewReady{id}` and stale ids are dropped. Reads are capped (`max_text_bytes`) and
  use regular-file checks (FIFOs and devices are never read: they block).
* **Text:** chroma v2 lexer by file name, then by content analysis; ANSI formatter sized to the terminal color depth.
* **Images:** Kitty graphics protocol only in M1. Detection, in order: `preview.images` config; a query (`a=q`
  followed by DA1, 200 ms timeout) written at startup before raw mode is claimed by the UI; the env hints
  (`TERM=xterm-ghostty`, `TERM_PROGRAM=ghostty`, `KITTY_WINDOW_ID`) are only a tie-breaker. If the terminal answers
  DA1 without answering the query, images are off. Images are sent as PNG (decoded and re-encoded with the standard
  library, size-capped) and placed within the popup rectangle; the popup close and any resize delete the image
  (`a=d`) so nothing is left on screen.
* **Fallback:** a facts panel (type, size, dimensions) and key `Q` runs `qlmanage -p <file>` as a separate window
  with a timeout; thumbnails through `qlmanage -t` are not used (reported hangs, research 03).
* Sixel and iTerm2 are not implemented (Ghostty documents neither); they are ROADMAP items.
* Port from lazymark: its `internal/image/kitty.go` (494 lines) as the starting point, adapted and tested here.

### Consequences

* Good: one tested protocol covers the target terminal; fallback is always usable.
* Bad: multiplexers may swallow graphics sequences; the probe must handle "no answer" and the user can force `off`.
* To measure in M1: popup open-to-image latency on a 12 MP photo, and capture evidence in Ghostty.
