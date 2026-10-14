package ui

import (
	"os"
	"path/filepath"
	"strings"
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
func TestActionIconsCreateFilesAndFoldersRefreshAndCollapse(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(e.root, "outside.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	click(actionRefresh)
	e.mustHave("outside.txt")
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
