# Markdown renders in a child process

* Status: accepted
* Date: 2026-10-19

## Context and problem statement

The preview renders Markdown with glamour, on top of goldmark. Neither takes a context, and the final security
reviews of the audit found inputs of a few hundred KB, from a folder you only look at, that run for minutes (nested
quotes, unclosed links, a table of thousands of columns), fill gigabytes, or overflow the stack (nested emphasis). A
stack overflow is a fatal error in Go: no recover, no deferred terminal restore. Limits guessed from the input (a
nesting depth) were bypassed by a fake fence and refused ordinary files.

## Decision

The render runs in a child process: lazyfinder starts its own executable again with `LAZYFINDER_RENDER_MARKDOWN` set,
writes the Markdown to its stdin and reads the rendered text. The `preview` package serves it from an `init`, so the
child renders before `main` (or a test binary's `TestMain`) runs. The child has a stack limit of 64 MB, ends itself
past 512 MB of heap or 4 s, and the parent kills it after 2 s, when the preview moves on, or past 8 MB of output. Any
failure shows the source, highlighted as Markdown.

## Consequences

* No Markdown can hang, exhaust or crash lazyfinder; the worst case is two seconds of one core and the source.
* Each rendered Markdown preview starts a process (about 20 ms); its stderr is discarded.
* Nothing else changes for the user: `preview.markdown: "raw"` still skips the render.
