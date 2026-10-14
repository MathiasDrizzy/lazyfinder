package ui

import (
	"os"
	"path/filepath"
	"strings"

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
	spans := e.m.actionSpans()
	if len(spans) != actionCount {
		t.Fatalf("want %d action icons at 40 columns, got %d", actionCount, len(spans))
	}
	click := func(i int) { e.click(spans[i].x, spans[i].y) }
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
