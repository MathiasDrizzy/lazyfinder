# Previews++: what ships, what waits

* Status: accepted (2026-10-13, milestone M5; requirement R-14)
* Date: 2026-10-13

## Context and problem statement

The ROADMAP lists for M5: Sixel and iTerm2 protocols, PDF first page, video and audio metadata, archive listing,
Markdown rendering and a hex view. Each is optional and some depend on the terminal or on tools that may not be
installed. The preview popup must stay fast, must never run a shell, and must never write to the user's files.

## Decision outcome

* **Hex view.** A file that is binary (a NUL in its first bytes, or a PDF, media or archive that could not get its own
  view) shows `hexdump -C` style rows of its first `preview.hex_bytes` bytes (default 4096). It replaces the old
  "Binary file" message.
* **Archive listing.** `.zip` (and `.jar`, `.docx`, `.xlsx`, `.pptx`), `.tar`, `.tar.gz` and `.tgz` list their
  entries with sizes, using only the standard library. Nothing is ever extracted. The list is capped at 500 lines,
  the total count is still shown, and names are stripped of control characters so a hostile archive cannot send
  terminal escapes. A damaged archive falls back to the hex view.
* **Markdown.** `.md` and `.markdown` are rendered with glamour v2 (the library lazymark already uses, so the look
  is the same in both tools) at the popup width. `preview.markdown: "raw"` shows the source; a file cut at
  `preview.max_text_bytes` is shown raw too, because half a document renders badly. `NO_COLOR` selects the
  uncolored style.
* **Media facts and PDF first page** use external tools behind two injectable function types (`preview.Probe`,
  `preview.Raster`): `mdls` for duration, dimensions and codecs; `pdftoppm` if present, else `sips`, for page one,
  drawn through the existing Kitty image path. Arguments are fixed argv (no shell), each call has a timeout, the
  temporary page lives in a private folder that only `preview.Discard` removes, and any failure falls back to the
  hex view. `preview.external_tools: "off"` never runs any of them.
* **Sixel and iTerm2 are not implemented.** The popup is drawn over the list and must survive redraws, scrolling and
  multiplexers. Kitty Unicode placeholders are text cells, so they compose with the popup and clear when it closes;
  Sixel and iTerm2 images are painted at the cursor and would need their own erase and compositing logic per
  terminal, which cannot be tested without those terminals. Where Kitty graphics are not available the popup offers
  Quick Look (`Q`), as before. This is revisited if a Sixel-only terminal is a requirement; it is left as a
  decision for the maintainer.

## Consequences

* Good: five of the six items work with no new protocol; every path has a unit test on temporary fixtures and the
  views have a tuiqa scenario (`previews-more.json`) with external tools off, so it is the same on any machine.
* Cost: one new dependency family (glamour v2, goldmark, their helpers) for Markdown. Accepted because the
  alternative is a worse hand-made renderer and a different look from lazymark.
* Cost: PDF and media depend on tools that may be missing; the fallback is the hex view, never an error.
* Not verified: `SystemProbe` and `SystemRaster` against real `mdls`, `pdftoppm` and `sips` output beyond the
  parser test and a manual run on the maintainer's machine.
