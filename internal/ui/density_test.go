package ui

import (
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/theme"
	"github.com/charmbracelet/x/ansi"
)

// lineOf is the index of the screen line holding word in its plain text.
func lineOf(t *testing.T, e *env, word string) int {
	t.Helper()
	for i, l := range strings.Split(raw(e), "\n") {
		if strings.Contains(ansi.Strip(l), word) {
			return i
		}
	}
	t.Fatalf("%q not on screen:\n%s", word, e.text())
	return -1
}

func rawAt(e *env, i int) string { return strings.Split(raw(e), "\n")[i] }

func (e *env) setDensity(d string) {
	e.m.deps.Cfg.Density = d
	e.m.ensureVisible()
}

func TestTheDefaultDensityIsWideLikeFinder(t *testing.T) {
	e := newEnv(t)
	e.m.deps.Cfg.Density = "comfortable"
	if e.m.rowH() != 2 {
		t.Fatalf("rowH = %d, want 2 lines per row", e.m.rowH())
	}
	if gap := lineOf(t, e, "data.xlsx") - lineOf(t, e, "beta"); gap != 2 {
		t.Fatalf("rows are %d lines apart, want 2", gap)
	}
}

func TestWideRowsHaveSoftStripesAndCompactOnesDoNot(t *testing.T) {
	e := newEnv(t)
	e.setDensity("comfortable")
	stripe := theme.BackgroundSeq(e.m.styles.stripeBg)
	beta, data := rawAt(e, lineOf(t, e, "beta")), rawAt(e, lineOf(t, e, "data.xlsx"))
	if !strings.Contains(beta, stripe) || strings.Contains(data, stripe) {
		t.Fatalf("the second row must be striped and the third not:\nbeta=%q\ndata=%q", beta, data)
	}
	if blank := rawAt(e, lineOf(t, e, "beta")+1); !strings.Contains(blank, stripe) {
		t.Fatalf("the blank line of a striped row must carry the stripe too: %q", blank)
	}
	e.setDensity("compact")
	if strings.Contains(raw(e), stripe) {
		t.Fatal("compact rows must not be striped")
	}
}

func TestTheSelectionIsAPillWithAMarginOnBothLinesOfARow(t *testing.T) {
	e := newEnv(t)
	e.setDensity("comfortable")
	i := lineOf(t, e, "alpha")
	first, second := rawAt(e, i), rawAt(e, i+1)
	if !strings.Contains(first, " \x1b[") || !strings.Contains(second, mochaSelectionBg) {
		t.Fatalf("the pill covers both lines of the row:\n%q\n%q", first, second)
	}
	plain := ansi.Strip(first)
	x := strings.Index(plain, "▸")
	if x < 0 || plain[x-len(" "):x] != " " {
		t.Fatalf("the pill must start one cell inside the panel: %q", plain)
	}
}

func TestNormalDensityIsOneLineWithStripes(t *testing.T) {
	e := newEnv(t)
	e.setDensity("normal")
	if e.m.rowH() != 1 {
		t.Fatalf("rowH = %d, want 1", e.m.rowH())
	}
	if !strings.Contains(rawAt(e, lineOf(t, e, "beta")), theme.BackgroundSeq(e.m.styles.stripeBg)) {
		t.Fatal("normal rows are striped")
	}
}
