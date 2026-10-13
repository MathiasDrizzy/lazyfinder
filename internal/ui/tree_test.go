package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// indentOf is the screen column where name starts in row (not the byte offset: the borders are wide characters).
func indentOf(row, name string) int { return ansi.StringWidth(row[:strings.Index(row, name)]) }

func TestPlusExpandsTheFolderInPlaceAndMinusCollapsesIt(t *testing.T) {
	e := newEnv(t)
	e.mustNotHave("inner.txt")
	e.clickText("alpha")
	e.key("+")
	e.mustHave("inner.txt")
	e.mustHave("deep")
	e.key("-")
	e.mustNotHave("inner.txt")
}

func TestChildrenAreIndentedAndTheMarkerShowsTheState(t *testing.T) {
	e := newEnv(t)
	e.clickText("alpha")
	e.key("+")
	if !strings.Contains(e.row("alpha"), "v") {
		t.Fatalf("an expanded folder shows v:\n%s", e.row("alpha"))
	}
	if indentOf(e.row("inner.txt"), "inner.txt") <= indentOf(e.row("alpha"), "alpha") {
		t.Fatalf("children are indented under their folder:\n%s\n%s", e.row("alpha"), e.row("inner.txt"))
	}
	if !strings.Contains(e.row("beta"), ">") {
		t.Fatalf("a collapsed folder keeps its > marker:\n%s", e.row("beta"))
	}
}

func TestExpandingKeepsTheSelectionOnTheFolder(t *testing.T) {
	e := newEnv(t)
	e.clickText("alpha")
	e.key("+")
	if sel, _ := e.m.selected(); sel.Name != "alpha" {
		t.Fatalf("selected %s, want alpha", sel.Name)
	}
	e.key("down")
	if sel, _ := e.m.selected(); sel.Name != "deep" {
		t.Fatalf("down goes into the children, selected %s", sel.Name)
	}
}

func TestClickingTheMarkerTogglesTheFolder(t *testing.T) {
	e := newEnv(t)
	_, y := e.find("alpha")
	e.click(e.m.listX()+2, y)
	e.mustHave("inner.txt")
	e.click(e.m.listX()+2, y)
	e.mustNotHave("inner.txt")
}

func TestMinusOnAChildCollapsesItsParentAndSelectsIt(t *testing.T) {
	e := newEnv(t)
	e.clickText("alpha")
	e.key("+")
	e.clickText("inner.txt")
	e.key("-")
	if sel, _ := e.m.selected(); sel.Name != "alpha" {
		t.Fatalf("selected %s, want alpha", sel.Name)
	}
	e.mustNotHave("inner.txt")
}

func TestFoldersExpandInsideFolders(t *testing.T) {
	e := newEnv(t)
	e.clickText("alpha")
	e.key("+")
	e.clickText("deep")
	e.key("+")
	e.mustHave("x.txt")
	if indentOf(e.row("x.txt"), "x.txt") <= indentOf(e.row("deep"), "deep") {
		t.Fatal("grandchildren are indented one more level")
	}
}

func TestExpansionSurvivesLeavingAndComingBack(t *testing.T) {
	e := newEnv(t)
	e.clickText("alpha")
	e.key("+")
	e.clickText("beta")
	e.key("enter")
	e.key("backspace")
	e.mustHave("inner.txt")
}

func TestExpandedFoldersAreWatchedAndRefreshLive(t *testing.T) {
	e, fw := liveEnv(t, "")
	e.clickText("alpha")
	e.key("+")
	alpha := filepath.Join(e.root, "alpha")
	if !fw.has(alpha) || !fw.has(e.root) {
		t.Fatalf("watched = %v, want the folder and alpha", fw.watched)
	}
	if err := os.WriteFile(filepath.Join(alpha, "agent.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.send(dirChangedMsg{dir: alpha})
	e.mustHave("agent.go")
	e.key("-")
	if fw.has(alpha) {
		t.Fatal("collapsing stops watching the folder")
	}
}

func TestEnterOnAChildFolderGoesIntoIt(t *testing.T) {
	e := newEnv(t)
	e.clickText("alpha")
	e.key("+")
	e.clickText("deep")
	e.key("enter")
	if e.m.dir() != filepath.Join(e.root, "alpha", "deep") {
		t.Fatalf("dir = %s", e.m.dir())
	}
}

func TestOpeningAChildFileUsesItsOpener(t *testing.T) {
	e := newEnv(t)
	e.clickText("alpha")
	e.key("+")
	e.clickText("inner.txt")
	e.key("enter")
	if len(e.run.calls) != 1 || e.run.calls[0][1] != filepath.Join(e.root, "alpha", "inner.txt") {
		t.Fatalf("calls = %v", e.run.calls)
	}
}

func TestSortingKeepsTheTree(t *testing.T) {
	e := newEnv(t)
	e.clickText("alpha")
	e.key("+")
	e.key("S")
	e.mustHave("inner.txt")
	if indentOf(e.row("inner.txt"), "inner.txt") <= indentOf(e.row("alpha"), "alpha") {
		t.Fatal("children stay under their folder after sorting")
	}
}

func TestAnEmptyFolderExpandsToNothing(t *testing.T) {
	e := newEnv(t)
	e.clickText("beta")
	e.key("+")
	if n := len(e.m.entries()); n != 4 { // alpha beta data.xlsx notes.txt
		t.Fatalf("entries = %d, want 4", n)
	}
}
