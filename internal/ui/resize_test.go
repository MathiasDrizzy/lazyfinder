package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
)

func (e *env) drag(x0, y0, x1, y1 int) {
	e.t.Helper()
	e.send(tea.MouseClickMsg{X: x0, Y: y0, Button: tea.MouseLeft})
	e.send(tea.MouseMotionMsg{X: (x0 + x1) / 2, Y: y1, Button: tea.MouseLeft})
	e.send(tea.MouseMotionMsg{X: x1, Y: y1, Button: tea.MouseLeft})
	e.send(tea.MouseReleaseMsg{X: x1, Y: y1, Button: tea.MouseLeft})
}

func (e *env) withConfigFile() string {
	d := e.m.deps
	d.ConfigPath = filepath.Join(e.home, "config.json")
	e.m = New(d)
	e.send(sizeMsg())
	return d.ConfigPath
}

func (e *env) savedConfig(path string) config.Config {
	e.t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		e.t.Fatal(err)
	}
	cfg, err := config.Parse(raw, e.home)
	if err != nil {
		e.t.Fatal(err)
	}
	return cfg
}

func (e *env) colSpec(c fsmodel.Column) colSpec {
	for _, s := range e.m.columns() {
		if s.col == c {
			return s
		}
	}
	e.t.Fatalf("column %v is not shown", c)
	return colSpec{}
}

func TestKeysResizeTheSidebar(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	start := e.m.sidebarW()
	e.key(">")
	if got := e.m.sidebarW(); got != start+2 {
		t.Fatalf("> widens by two: %d -> %d", start, got)
	}
	e.key("<")
	e.key("<")
	e.key("<")
	if got := e.m.sidebarW(); got != start-4 {
		t.Fatalf("< narrows: %d", got)
	}
	if cfg := e.savedConfig(path); cfg.SidebarWidth != start-4 {
		t.Fatalf("saved %d", cfg.SidebarWidth)
	}
	for range 40 {
		e.key("<")
	}
	if got := e.m.sidebarW(); got != config.MinSidebar {
		t.Fatalf("the sidebar stops at %d, is %d", config.MinSidebar, got)
	}
	for range 60 {
		e.key(">")
	}
	if got := e.m.sidebarW(); got > e.m.w/2 {
		t.Fatalf("the sidebar never takes more than half the window: %d", got)
	}
}

func TestDraggingTheSidebarBorderResizesItAndSavesOnRelease(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	b := e.m.sidebarW() - 1 // the sidebar's own right border
	e.drag(b, 10, b+8, 10)
	if got := e.m.sidebarW(); got != 30 {
		t.Fatalf("sidebar = %d, want 30", got)
	}
	if cfg := e.savedConfig(path); cfg.SidebarWidth != 30 {
		t.Fatalf("saved %d", cfg.SidebarWidth)
	}
	e.mustHave("alpha") // the list still draws
}

func TestDraggingAColumnBorderResizesThatColumn(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	e.send(tea.WindowSizeMsg{Width: 150, Height: 40}) // room for the columns to grow
	size := e.colSpec(fsmodel.ColSize)
	bx := e.m.listX() + size.x + size.w // the gap right of Size
	e.drag(bx, headerRow, bx-3, headerRow)
	if got := e.colSpec(fsmodel.ColSize).w; got != size.w-3 {
		t.Fatalf("Size = %d, want %d", got, size.w-3)
	}
	if cfg := e.savedConfig(path); cfg.ColumnWidths["size"] != size.w-3 {
		t.Fatalf("saved %v", cfg.ColumnWidths)
	}
	// the border of Name moves the column on its right the other way round
	name := e.colSpec(fsmodel.ColName)
	mod := e.colSpec(fsmodel.ColModified)
	nx := e.m.listX() + name.x + name.w
	e.drag(nx, headerRow, nx-4, headerRow)
	if got := e.colSpec(fsmodel.ColModified).w; got != mod.w+4 {
		t.Fatalf("Modified = %d, want %d", got, mod.w+4)
	}
	if e.m.sort.By != fsmodel.ColName {
		t.Fatal("a drag on a border must not sort")
	}
}

func TestDoubleClickOnAColumnBorderFitsItToItsContent(t *testing.T) {
	e := newEnv(t)
	e.send(tea.WindowSizeMsg{Width: 150, Height: 40}) // room for the columns to grow
	e.m.deps.Cfg.ColumnWidths = map[string]int{"kind": 40}
	e.send(tea.WindowSizeMsg{Width: 150, Height: 40})
	kind := e.colSpec(fsmodel.ColKind)
	bx := e.m.listX() + kind.x + kind.w
	e.click(bx, headerRow)
	e.click(bx, headerRow)
	want := len("Excel Spreadsheet") // the longest kind in the fixture
	if got := e.colSpec(fsmodel.ColKind).w; got != want {
		t.Fatalf("Kind = %d, want %d", got, want)
	}
}

func TestKeysResizeTheSortedColumn(t *testing.T) {
	e := newEnv(t)
	e.send(tea.WindowSizeMsg{Width: 150, Height: 40}) // room for the columns to grow
	e.key("s")                                        // sort by the next column: Date Modified
	col := e.m.sort.By
	before := e.colSpec(col).w
	e.key("ctrl+right")
	if got := e.colSpec(col).w; got != before+2 {
		t.Fatalf("ctrl+right widens: %d -> %d", before, got)
	}
	e.key("ctrl+left")
	e.key("ctrl+left")
	if got := e.colSpec(col).w; got != before-2 {
		t.Fatalf("ctrl+left narrows: %d", got)
	}
}

func TestDraggingThePreviewBorderOfTheColumnViewResizesIt(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	e.key("M")
	g := e.m.millerGeom()
	bx := e.m.listX() + g.parentW + 1 + g.centerW
	e.drag(bx, 10, bx-6, 10)
	if got := e.m.millerGeom().rightW; got != g.rightW+6 {
		t.Fatalf("preview = %d, want %d", got, g.rightW+6)
	}
	if cfg := e.savedConfig(path); cfg.PreviewWidth != g.rightW+6 {
		t.Fatalf("saved %d", cfg.PreviewWidth)
	}
}
