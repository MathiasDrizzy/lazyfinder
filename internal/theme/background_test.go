package theme

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestPaintBackgroundCoversEveryLineAndSurvivesResets(t *testing.T) {
	bg := lipgloss.Color("#102030")
	out := PaintBackground("plain\n\x1b[31mred\x1b[0m tail", bg)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("line count changed: %q", out)
	}
	const set = "\x1b[48;2;16;32;48m"
	for i, l := range lines {
		if !strings.HasPrefix(l, set) {
			t.Errorf("line %d does not start with the background: %q", i, l)
		}
	}
	if !strings.Contains(lines[1], "\x1b[0m"+set+" tail") {
		t.Errorf("the background must come back after a reset: %q", lines[1])
	}
}

func TestPaintBackgroundKeepsAnExplicitBackground(t *testing.T) {
	out := PaintBackground("\x1b[48;2;1;2;3mX\x1b[0m", lipgloss.Color("#102030"))
	if strings.Contains(out, "\x1b[48;2;1;2;3m\x1b[48;2;16;32;48m") {
		t.Errorf("a cell with its own background must not be repainted: %q", out)
	}
}

// Text that carries no color of its own takes the theme's text color, so a light theme stays readable on a terminal
// whose own foreground is light (order 2026-10-13-06: the file names of the latte capture were pale grey).
func TestPaintBackgroundWithATextColorKeepsPlainTextReadable(t *testing.T) {
	out := PaintBackground("plain\n\x1b[31mred\x1b[0m tail\x1b[39m end", lipgloss.Color("#102030"), lipgloss.Color("#a0b0c0"))
	const fg = "\x1b[38;2;160;176;192m"
	lines := strings.Split(out, "\n")
	for i, l := range lines {
		if !strings.Contains(l, fg) || strings.Index(l, fg) > len("\x1b[48;2;16;32;48m") {
			t.Errorf("line %d does not start with the text color: %q", i, l)
		}
	}
	if !strings.Contains(lines[1], "\x1b[0m\x1b[48;2;16;32;48m"+fg+" tail") || !strings.Contains(lines[1], "\x1b[39m"+fg+" end") {
		t.Errorf("the text color must come back after a reset and after 39: %q", lines[1])
	}
}
