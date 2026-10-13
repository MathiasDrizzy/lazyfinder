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
