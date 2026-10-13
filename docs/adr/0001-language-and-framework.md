# Go and Bubble Tea v2

* Status: proposed
* Date: 2026-10-13

## Context and problem statement

lazyfinder needs a language and TUI framework. It must handle mouse input, overlays (the preview popup), inline
images, a file watcher, and be testable with the real binary in a pty (ED-25). It should share its nature with
lazymark, Mathias's other TUI.

## Decision drivers

* Mouse as a first-class input.
* Reuse of proven code and test tooling from lazymark and cerebro's `tuiqa` (Go).
* Ecosystem for file watching, syntax highlighting, Kitty graphics.
* Responsiveness on directories of 10 000+ entries.

## Considered options

* Go + Bubble Tea v2 (`charm.land/bubbletea/v2`, lipgloss v2, bubbles v2).
* Go + tcell/tview.
* Rust + Ratatui (the yazi route).

## Decision outcome

Chosen option: **Go + Bubble Tea v2**, with chroma v2 (`github.com/alecthomas/chroma/v2`) for highlighting and
`github.com/fsnotify/fsnotify` for watching (ADR 0003). Go 1.27.2, pinned with `go` and `toolchain` in `go.mod`.

Evidence:

* Listing and `stat` of 10 000 entries took 27 ms in Go (research 02), so Go is not a bottleneck and Rust's speed
  advantage buys nothing visible for this workload.
* lazymark already runs the same stack in production (`bubbletea/v2 v2.0.10`, `lipgloss/v2 v2.0.6`,
  `bubbles/v2 v2.2.1`, `chroma/v2 v2.27.0`) and has a 494-line Kitty image module and mouse hit-testing to port.
* Bubble Tea v2 reports click, release, wheel and motion (`tea.Mouse*Msg`) through `View()` (research 03).
* The shared `tuiqa` harness and the Ghostty capture script are Go/language-agnostic, but the testing idioms in the
  standard (`teatest`, golden files) are Go-native.
* The agent's cost to write and review Go on this stack is the lowest of the three (existing templates).

Risks and mitigations:

* No explicit "stable" announcement for Bubble Tea v2 was found. Mitigation: pin exact versions in `go.mod`, let
  Dependabot propose bumps with cooldown, and keep all framework calls inside `internal/ui` so a replacement is local.
* Chroma v3 is alpha. Stay on v2.27.0.

### Consequences

* Good: fastest path to a usable M1; one toolchain across Mathias's TUIs.
* Bad: Bubble Tea's string-based rendering needs care for overlays; image placement needs raw escape output outside
  the cell grid (ADR 0006).
