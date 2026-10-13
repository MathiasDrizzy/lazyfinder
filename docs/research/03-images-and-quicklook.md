# Research 03: terminal images, Quick Look and the terminal stack

Consulted 2026-10-13. Items marked NOT VERIFIED had no primary source read.

## Image protocols

- **Kitty graphics protocol:** detection is a query action (`a=q`) followed by a DA1 request; a DA1 reply without a
  query reply means no support. Unicode placeholders use U+10EEEE.
  <https://sw.kovidgoyal.net/kitty/graphics-protocol/>
- **Ghostty** documents support for the Kitty graphics protocol; it does not document Sixel or iTerm2 inline images.
  <https://ghostty.org/docs/features> (Sixel in Ghostty: community reports of no support, not verified.)
- **iTerm2:** `OSC 1337 ; File=...`; no query of its own. <https://iterm2.com/documentation-images.html>
- **Detection order used by yazi** (read from source, summary not confirmed): exact `TERM`, exact `TERM_PROGRAM`,
  presence variables (`KITTY_WINDOW_ID`, `GHOSTTY_RESOURCES_DIR`, ...), then a terminal probe with a timeout.
- **This machine's session:** `TERM_PROGRAM=herdr` (multiplexer), `TERM=xterm-256color`, so environment-variable
  detection alone is unreliable here; the query-based probe is required and must tolerate multiplexers.
- Reusable code: `lazymark/internal/image/kitty.go` (494 lines) already implements Kitty output for Mathias's other TUI.
  Reuse by copying with attribution inside the same owner's repos, not by importing across repos (each agent owns its repo).

## Quick Look

- `qlmanage -p` opens a separate preview window; `qlmanage -t -o <dir> -s <px> <file>` writes a PNG thumbnail.
  <https://ss64.com/mac/qlmanage.html>; thumbnail usage: <https://alexwlchan.net/2020/using-qlmanage-to-create-thumbnails-on-macos>
- Reported problems: `-m` broken on Sequoia, hangs on images with many alpha channels (community, not confirmed).
- Conclusion: Quick Look is a **fallback** when the terminal cannot show images, started as an external window on
  request. It is never the primary preview, and its thumbnail path needs a timeout.

## Stack

- Bubble Tea v2 import path `charm.land/bubbletea/v2`; mouse is configured in `View()` (`tea.View.MouseMode`), messages
  are `tea.MouseClickMsg`, `MouseReleaseMsg`, `MouseWheelMsg`, `MouseMotionMsg`; key messages are `tea.KeyPressMsg`.
  Lip Gloss v2 and Bubbles v2 are required together. Releases seen 2026-10-13: Bubble Tea v2.1.0 (Oct 8), Lip Gloss
  v2.0.6, Bubbles v2.2.1. An explicit "stable" announcement was NOT found; cadence of patch releases only.
  <https://github.com/charmbracelet/bubbletea>
- tcell v3.6.0 (Oct 11) is the low-level alternative; tview state NOT VERIFIED.
- Syntax highlighting: chroma. README says v2 but shows `/v3`; "Latest" is `v3.0.0-alpha.5`, the stable line is v2.27.0.
  Decision: stay on v2 (lazymark already runs v2.27.0 in production). <https://github.com/alecthomas/chroma/releases>
- File kind ("Kind" column): `mdls -name kMDItemContentType` per file is slow (cost NOT DOCUMENTED, one process per
  file) and macOS-only. A built-in extension table, with directory/symlink/executable special cases, is deterministic
  and testable (ADR 0002).
