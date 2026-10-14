package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// Mathias, 2026-10-13, comparing with the herdr sidebar he has installed: the sidebar mode must look like it. These
// tests pin the layout of herdr-sidebar's explorer (explorer_app.rs, draw and draw_activity_bar).

// The tab bar is three rows tall: the icons sit on the middle row and the chip of the active tab reaches into the rows
// above and below with half blocks, a tall button with room around the icon.
func TestTheSidebarTabBarIsThreeRowsTallLikeHerdr(t *testing.T) {
	e := projectEnv(t, 40)
	s := e.screen()
	_, spans := e.m.projectTabsLine()
	files := spans[tabFiles]
	top := []rune(s[0])
	bottom := []rune(s[2])
	for x := files.x; x < files.x+files.w; x++ {
		if top[x] != '▄' || bottom[x] != '▀' {
			t.Fatalf("the active tab must reach rows 0 and 2 with half blocks at x=%d:\n%s", x, e.text())
		}
	}
	if !strings.Contains(s[1], "\uf07b") {
		t.Fatalf("the icons go on the middle row:\n%s", e.text())
	}
	if !strings.HasPrefix(s[3], " PROJ") {
		t.Fatalf("the project name heads the tree on row 3:\n%s", e.text())
	}
}

// Every row of a tab button takes the click, as in herdr (hits_activity_button).
func TestEachRowOfATabButtonOpensIt(t *testing.T) {
	for _, y := range []int{0, 1, 2} {
		e := projectEnv(t, 40)
		_, spans := e.m.projectTabsLine()
		e.click(spans[tabGit].x+1, y)
		if !e.m.scmOn() {
			t.Fatalf("a click on row %d of the Git tab must open it", y)
		}
	}
}

// herdr frames the pane itself, so the sidebar draws no frame of its own: no border and no counter.
func TestTheSidebarHasNoFrameOfItsOwn(t *testing.T) {
	e := projectEnv(t, 40)
	for _, c := range []string{"╭", "╰", "│", " of "} {
		e.mustNotHave(c)
	}
}

// A tree row is the indent, a dimmed triangle for folders, the icon and the name, from the first column; no pointer
// and no ">" marker. The name keeps the plain color, like herdr and VS Code.
func TestTheSidebarRowsAreTheIndentTheTriangleTheIconAndTheName(t *testing.T) {
	e := projectEnv(t, 40)
	x, y := e.find("alpha")
	line := e.screen()[y]
	if !strings.HasPrefix(line, "▸ ") {
		t.Fatalf("a closed folder starts with ▸ at the first column: %q", line)
	}
	if x != 4 {
		t.Fatalf("the name of a top-level entry starts after the triangle and the icon (x=4), got %d: %q", x, line)
	}
	e.click(x, y)
	if !strings.HasPrefix(e.screen()[y], "▾ ") {
		t.Fatalf("an open folder starts with ▾: %q", e.screen()[y])
	}
	_, dy := e.find("deep")
	if l := e.screen()[dy]; !strings.HasPrefix(l, "  ▸ ") {
		t.Fatalf("a nested folder is indented two cells per level: %q", l)
	}
	for _, l := range e.screen()[4:] {
		if strings.HasPrefix(l, "> ") || strings.Contains(l, " > ") {
			t.Fatalf("no > marker in the sidebar tree: %q", l)
		}
	}
}

// The selected row keeps the colors of its icon, as herdr's selection only sets the background.
func TestTheSelectedRowKeepsTheColorOfItsIcon(t *testing.T) {
	e := projectEnv(t, 40)
	raw := strings.Split(e.m.View().Content, "\n")
	_, y := e.find("alpha")
	if e.m.entries()[e.m.sel].Name != "alpha" {
		t.Skip("the first row is not alpha")
	}
	if strings.Count(raw[y], "\x1b[") < 3 || ansi.Strip(raw[y]) == raw[y] {
		t.Fatalf("the selected row must keep its own colors: %q", raw[y])
	}
	if !strings.Contains(raw[y], "38;2;") {
		t.Fatalf("the icon of the selected row keeps its color: %q", raw[y])
	}
}

// The last row of the list takes the click (it used to be one row short of the bottom border).
func TestTheLastVisibleRowOfTheListTakesTheClick(t *testing.T) {
	e := newEnv(t)
	for i := 0; i < 40; i++ {
		e.touch("f"+string(rune('a'+i%26))+string(rune('a'+i/26))+".txt", "x")
	}
	e.m.reload()
	last := e.m.firstRow() + e.m.listRows() - 1
	e.click(e.m.listX()+8, last)
	if e.m.sel != e.m.top+e.m.listRows()-1 {
		t.Fatalf("a click on the last visible row (y=%d) must select it: sel=%d top=%d rows=%d", last, e.m.sel, e.m.top, e.m.listRows())
	}
}
