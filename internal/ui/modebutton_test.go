package ui

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func (e *env) move(x, y int) { e.send(tea.MouseMotionMsg{X: x, Y: y}) }

// order 07, point M1: a button switches between the normal window and the sidebar mode, in both of them.
func TestTheModeButtonSwitchesToTheSidebarModeAndBack(t *testing.T) {
	e := newEnv(t)
	x := e.m.w - e.m.gearW() - modeBtnW + 1
	e.click(x, 0)
	if !e.m.projectMode() {
		t.Fatal("clicking the mode button must enter the sidebar mode")
	}
	e.click(e.m.w-e.m.gearW()-modeBtnW+1, 0)
	if e.m.projectMode() {
		t.Fatal("clicking it again must bring the window back")
	}
}

// order 07, point M5: action icons in the header of the tree.
func TestActionIconsCreateFilesAndFoldersAndCollapse(t *testing.T) {
	e := projectEnv(t, 40)
	e.move(5, 5) // the buttons appear while the pointer is over the pane
	spans := e.m.actionSpans()
	if len(spans) != actionCount {
		t.Fatalf("want %d action icons at 40 columns, got %d", actionCount, len(spans))
	}
	click := func(i int) { e.move(spans[i].x, spans[i].y); e.click(spans[i].x, spans[i].y) } // the pointer arrives before the click
	click(actionNewFile)
	if e.m.prompt.kind != promptNewFile {
		t.Fatalf("the first icon asks for a file name, prompt = %v", e.m.prompt.kind)
	}
	e.key("enter")
	if _, err := os.Stat(filepath.Join(e.root, "alpha", "untitled file")); err != nil { // the selected row is alpha
		t.Fatalf("the new file goes in the selected folder: %v", err)
	}
	click(actionNewFolder)
	if e.m.prompt.kind != promptNewFolder {
		t.Fatal("the second icon asks for a folder name")
	}
	e.key("esc")
	e.key("right") // opens alpha
	if !e.m.expanded[filepath.Join(e.root, "alpha")] {
		t.Fatal("setup: alpha should be open")
	}
	click(actionCollapse)
	if len(e.m.expanded) != 0 {
		t.Fatal("the fourth icon collapses everything")
	}
}

// order 07, point N: the row under the pointer is highlighted, and it stops when a key is pressed.
func TestHoverPaintsTheRowUnderThePointerAndKeysTakeItOff(t *testing.T) {
	e := newEnv(t)
	e.setDensity("compact")
	i := lineOf(t, e, "beta")
	before := rawAt(e, i)
	x, y := e.find("beta")
	e.move(x, y)
	if after := rawAt(e, i); after == before || !strings.Contains(after, "48;2;") {
		t.Fatalf("hovering must change the row:\n%q\n%q", before, after)
	}
	e.key("down")
	if _, ok := e.m.hoverRect(); ok {
		t.Fatal("a key press takes the hover off")
	}
}

func TestTheTooltipNamesTheIconUnderThePointer(t *testing.T) {
	e := projectEnv(t, 40)
	e.move(5, 5)
	a := e.m.actionSpans()[actionNewFolder]
	e.move(a.x, a.y)
	if !strings.Contains(e.text(), "New folder: N") {
		t.Fatalf("no tooltip:\n%s", e.text())
	}
	e.move(e.m.w-e.m.gearW()-modeBtnW+1, 0)
	if !strings.Contains(e.text(), "Leave the sidebar mode: ctrl+b") {
		t.Fatalf("no tooltip for the mode button:\n%s", e.text())
	}
}

func TestNoMouseTurnsHoverOff(t *testing.T) {
	e := newEnv(t)
	e.m.mouse = false
	x, y := e.find("beta")
	e.move(x, y)
	if _, ok := e.m.hoverRect(); ok {
		t.Fatal("--no-mouse has no hover")
	}
}

// Mathias, 2026-10-13: the buttons were small, uneven and "some do not work": every button is three cells wide, the
// glyph in the middle, and a click on any of its three cells runs it.
func TestEveryCellOfAnActionButtonRunsIt(t *testing.T) {
	for _, w := range []int{30, 40, 80} {
		e := projectEnv(t, w)
		e.move(5, 5)
		for _, sp := range e.m.actionSpans() {
			if sp.w != 3 {
				t.Fatalf("width %d: button %d is %d cells wide, want 3", w, sp.idx, sp.w)
			}
			for dx := 0; dx < sp.w; dx++ {
				if z := e.m.hit(sp.x+dx, sp.y); z.kind != zoneAction || z.idx != sp.idx {
					t.Fatalf("width %d: cell %d of button %d hits %+v", w, dx, sp.idx, z)
				}
			}
		}
	}
}

func TestThereIsNoRefreshButtonBecauseTheListUpdatesByItself(t *testing.T) {
	e := projectEnv(t, 40)
	e.move(5, 5)
	for i := range e.m.actionDefs() {
		if e.m.actionLabel(i) == "Actualizar: r" || e.m.actionLabel(i) == "Refresh: r" {
			t.Fatal("the refresh button is gone")
		}
	}
}

func TestTheTabsShowOnlyTheirIconAndEachHasATooltip(t *testing.T) {
	e := projectEnv(t, 36)
	for _, name := range []string{"Archivos", "Buscar", "Files", "Search"} {
		e.mustNotHave(name + " ") // a tab label, not the footer hint
	}
	line, spans := e.m.projectTabsLine()
	if got := ansi.StringWidth(line); got != 3*5 {
		t.Fatalf("three square buttons of five cells, got %d cells", got)
	}
	for i, s := range spans {
		if s.w != 5 {
			t.Fatalf("tab %d is %d cells wide, want 5", i, s.w)
		}
		e.move(s.x+2, 0)
		if tip := e.m.tooltip(); strings.TrimSpace(tip) == "" {
			t.Fatalf("tab %d has no tooltip", i)
		}
	}
}

func TestAClickAnywhereOnAFolderRowOpensAndClosesItInTheTree(t *testing.T) {
	e := projectEnv(t, 40)
	alpha := filepath.Join(e.root, "alpha")
	x, y := e.find("alpha")
	e.click(x+3, y) // on the name, far from the > marker
	if !e.m.expanded[alpha] {
		t.Fatal("one click on the name opens the folder")
	}
	x, y = e.find("alpha")
	e.click(x, y)
	if e.m.expanded[alpha] {
		t.Fatal("another click closes it")
	}
}

// Mathias, 2026-10-13: "cuando se achique el cuadro salte de un modo a otro automáticamente y al revés también".
func TestTheModeChangesByItselfWhenTheWindowIsResizedAcrossTheLimits(t *testing.T) {
	e := newEnv(t)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 30})
	if e.m.projectMode() {
		t.Fatal("setup: a wide window starts as the normal window")
	}
	e.send(tea.WindowSizeMsg{Width: 70, Height: 30})
	if e.m.projectMode() {
		t.Fatal("70 columns still fit the normal window")
	}
	e.send(tea.WindowSizeMsg{Width: 50, Height: 30})
	if !e.m.projectMode() {
		t.Fatal("below 60 columns the window becomes the sidebar")
	}
	e.send(tea.WindowSizeMsg{Width: 70, Height: 30})
	if !e.m.projectMode() {
		t.Fatal("between 60 and 80 the sidebar stays: no flipping on every cell of a drag")
	}
	e.send(tea.WindowSizeMsg{Width: 100, Height: 30})
	if e.m.projectMode() {
		t.Fatal("from 80 columns up it is the normal window again")
	}
}

func TestTheModeTheProgramStartedInStaysUntilTheFirstResizeAcrossALimit(t *testing.T) {
	e := projectEnv(t, 120) // the sidebar mode in a wide pane, as asked on the command line
	if !e.m.projectMode() {
		t.Fatal("setup")
	}
	e.send(tea.WindowSizeMsg{Width: 110, Height: 24})
	if !e.m.projectMode() {
		t.Fatal("a resize that crosses no limit changes nothing")
	}
}

func TestAWindowThatStartsNarrowerThanTheLimitIsTheSidebar(t *testing.T) {
	e := newEnv(t)
	d := e.m.deps
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 44, Height: 26})
	if !e.m.projectMode() {
		t.Fatal("a 44-column pane has no room for the columns: it starts as the sidebar")
	}
	e.mustNotHave("too small")
}

// Mathias, 2026-10-13: "los recuadros aún están descuadrados": the glyph of a button must be drawn in the middle
// cell of the cells that the hover paints and the click hits, in every width and with or without a branch.
func TestEachGlyphIsDrawnInTheMiddleCellOfItsButton(t *testing.T) {
	for _, withGit := range []bool{false, true} {
		for w := 90; w <= 200; w += 7 {
			var e *env
			if withGit {
				e = projectEnv(t, w)
				e.key("ctrl+b") // back to the normal window
			} else {
				e = newEnv(t)
			}
			e.send(tea.WindowSizeMsg{Width: w, Height: 30})
			if e.m.projectMode() {
				t.Fatalf("setup: width %d must be the normal window", w)
			}
			line := e.screen()[0]
			for _, sp := range e.m.actionSpans() {
				if got := ansi.Strip(ansi.Cut(line, sp.x+1, sp.x+2)); got != e.m.actionGlyph(sp.idx) {
					t.Fatalf("git=%v width %d: cell %d holds %q, button %d draws %q (line %q)", withGit, w, sp.x+1, got, sp.idx, e.m.actionGlyph(sp.idx), ansi.Strip(line))
				}
			}
		}
	}
}

// The glyph of every button of the top row is drawn on cells that the click hits and the hover paints as that very
// button: the gear took two cells on the screen but four in the zones, which put every zone left of it two cells off.
func TestEveryTopRowGlyphLiesInsideItsOwnZone(t *testing.T) {
	check := func(label string, e *env, w int) {
		t.Helper()
		line := ansi.Strip(e.screen()[0])
		glyphs := []struct {
			name string
			g    string
			kind zoneKind
		}{
			{"gear", e.m.gearText(), zoneSettings},
			{"mode", e.m.modeGlyph(), zoneMode},
		}
		if !e.m.projectMode() {
			glyphs = append(glyphs, struct {
				name string
				g    string
				kind zoneKind
			}{"search", e.m.lensGlyph(), zoneSearch})
		}
		for _, g := range glyphs {
			i := strings.LastIndex(line, g.g)
			if i < 0 {
				t.Fatalf("%s width %d: no %s glyph in %q", label, w, g.name, line)
			}
			x := ansi.StringWidth(line[:i])
			if z := e.m.hit(x, 0); z.kind != g.kind {
				t.Fatalf("%s width %d: the %s glyph at cell %d hits %+v (line %q)", label, w, g.name, x, z, line)
			}
			if r, ok := hoverAt(e, x, 0); !ok || x < r.x || x >= r.x+r.w {
				t.Fatalf("%s width %d: the hover of the %s glyph at cell %d is %+v", label, w, g.name, x, r)
			}
		}
	}
	for _, w := range []int{60, 70, 80, 90, 100, 120, 160} {
		e := newEnv(t)
		e.send(tea.WindowSizeMsg{Width: w, Height: 30})
		if w < autoSidebarW || e.m.projectMode() {
			continue
		}
		check("window", e, w)
	}
	for _, w := range []int{24, 30, 36, 44, 60} {
		check("sidebar", projectEnv(t, w), w)
	}
}

func hoverAt(e *env, x, y int) (hoverRect, bool) {
	e.move(x, y)
	return e.m.hoverRect()
}

// Mathias, 2026-10-13: the buttons of the tree "do not disappear like in the sidebar": like the herdr sidebar they
// show while the pointer is active over the pane and fade a few seconds after the last movement.
func TestTheTreeButtonsAppearWithThePointerAndFadeAfterItStops(t *testing.T) {
	e := projectEnv(t, 40)
	if n := len(e.m.actionSpans()); n != 0 {
		t.Fatalf("without a pointer the header has no buttons, got %d", n)
	}
	e.move(5, 5)
	if len(e.m.actionSpans()) != actionCount {
		t.Fatal("the pointer over the pane shows the buttons")
	}
	if len(e.woke) != 1 || e.woke[0] != actionsLinger {
		t.Fatalf("a wake-up must be asked for to fade them, got %v", e.woke)
	}
	e.now = e.now.Add(actionsLinger - time.Second)
	e.send(actionsFadeMsg{})
	if len(e.m.actionSpans()) != actionCount {
		t.Fatal("still inside the linger")
	}
	e.now = e.now.Add(2 * time.Second)
	e.send(actionsFadeMsg{})
	if len(e.m.actionSpans()) != 0 {
		t.Fatal("after the linger the buttons are gone")
	}
	e.move(6, 5)
	if len(e.m.actionSpans()) != actionCount {
		t.Fatal("moving again brings them back")
	}
	e.key("down")
	if len(e.m.actionSpans()) != 0 {
		t.Fatal("a key press hides them: the keyboard takes over")
	}
}

func TestTheWindowToolbarKeepsItsButtonsAlways(t *testing.T) {
	e := newEnv(t)
	if len(e.m.actionSpans()) != actionCount {
		t.Fatal("the toolbar of the window is not a hover row")
	}
}

func TestTheTreeHeaderAndTheBorderDoNotRepeatTheProjectName(t *testing.T) {
	e := projectEnv(t, 40)
	n := 0
	for _, l := range e.screen() {
		if strings.Contains(strings.ToLower(l), "proj") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("the project name appears %d times, want once:\n%s", n, e.text())
	}
}
