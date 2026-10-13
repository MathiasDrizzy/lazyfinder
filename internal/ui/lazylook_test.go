package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// cells is a screen row as runes, so a column index is a cell.
func (e *env) cells(y int) []rune { return []rune(e.screen()[y]) }

func (e *env) sidebarRow(y int) string {
	r := e.cells(y)
	return string(r[1 : e.m.sidebarW()-1])
}

// Order 2026-10-13-06, point 1: a divider line between Favorites and Locations.
func TestSidebarHasADividerBetweenFavoritesAndLocations(t *testing.T) {
	e := newEnv(t)
	_, y := e.find("Locations")
	if got := strings.TrimSpace(e.sidebarRow(y - 1)); got == "" || strings.Trim(got, "─") != "" {
		t.Fatalf("the line above Locations must be a divider, got %q\n%s", got, e.text())
	}
}

func TestDividerIsNotASelectableOrClickableDestination(t *testing.T) {
	e := newEnv(t)
	_, y := e.find("Locations")
	before := e.m.dir()
	e.click(3, y-1)
	if e.m.dir() != before {
		t.Fatalf("clicking the divider moved to %s", e.m.dir())
	}
}

// Point 4: a │ between the columns, in the header and in every row, and dragging it still resizes.
func (e *env) listCells(y int) string {
	r := e.cells(y)
	return string(r[e.m.listX() : e.m.listX()+e.m.listW()])
}

func TestColumnsAreSeparatedByABarLikeFinder(t *testing.T) {
	e := newEnv(t)
	n := len(e.m.columns()) - 1
	for _, y := range []int{headerRow, firstRow, firstRow + 2} {
		if got := strings.Count(e.listCells(y), "│"); got != n {
			t.Fatalf("row %d has %d separators, want %d:\n%s", y, got, n, e.text())
		}
	}
}

func TestColumnSeparatorsCanBeSwitchedOffInSettings(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	e.key(",")
	e.pickSetting("Column separators")
	e.key("right")
	if e.m.deps.Cfg.ColumnSeparators || e.savedConfig(path).ColumnSeparators {
		t.Fatal("the setting must switch separators off and save it")
	}
	e.key("esc")
	if strings.Contains(e.listCells(headerRow), "│") {
		t.Fatalf("separators still drawn:\n%s", e.text())
	}
}

func TestDraggingASeparatorResizesTheColumn(t *testing.T) {
	e := newEnv(t)
	s := e.m.columns()[1]
	bar := e.m.listX() + s.x + s.w + 1 // the │ in the gap right of the first optional column
	if e.cells(headerRow)[bar] != '│' {
		t.Fatalf("expected a separator at x=%d: %q", bar, e.listCells(headerRow))
	}
	before := e.m.colWidth(s.col)
	e.drag(bar, headerRow, bar-3, headerRow) // the name has no room left to grow into
	if got := e.m.colWidth(s.col); got != before-3 {
		t.Fatalf("width %d, want %d", got, before-3)
	}
}

// Points 2 and 3: icon size and a bigger gear, one setting.
func (e *env) pickSetting(label string) {
	e.t.Helper()
	for i := 0; i < settingCount; i++ {
		if e.m.settingLabel(e.m.set.sel) == i18n.T(label, label) || e.m.settingLabel(e.m.set.sel) == label {
			return
		}
		e.key("down")
	}
	e.t.Fatalf("no setting called %q", label)
}

func TestIconSizeIsASettingThatSavesAndShowsItsValue(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	e.key(",")
	e.pickSetting("Icon size")
	e.mustHave("‹ medium ›")
	e.key("right")
	if e.m.deps.Cfg.IconSize != "large" || e.savedConfig(path).IconSize != "large" {
		t.Fatalf("icon size %q", e.m.deps.Cfg.IconSize)
	}
	e.key("right")
	if e.m.deps.Cfg.IconSize != "small" {
		t.Fatal("the value wraps around")
	}
}

func TestGearGrowsWithTheIconSizeAndKeepsAComfortableClickZone(t *testing.T) {
	width := map[string]int{}
	for _, size := range config.IconSizes {
		e := newEnv(t)
		e.m.deps.Cfg.IconSize = size
		e.send(sizeMsg())
		w := e.m.gearW()
		width[size] = w
		for _, x := range []int{e.m.w - 1, e.m.w - w} {
			e2 := e
			e2.click(x, 0)
			if !e2.m.set.open {
				t.Fatalf("%s: a click on x=%d of the gear (width %d) must open Settings", size, x, w)
			}
			e2.key("esc")
		}
		e.click(e.m.w-w-1, 0)
		if e.m.set.open {
			t.Fatalf("%s: the zone is wider than the gear", size)
		}
	}
	if !(width["small"] < width["medium"] && width["medium"] < width["large"]) {
		t.Fatalf("the gear must grow with the size: %v", width)
	}
}

func TestLargeIconsGetTwoLineRowsAndSmallOnesDoNot(t *testing.T) {
	e := newEnv(t)
	e.m.deps.Cfg.IconSize = "small"
	if e.m.rowH() != 1 {
		t.Fatal("small icons keep one-line rows")
	}
	e.m.deps.Cfg.IconSize = "large"
	if e.m.rowH() != 2 {
		t.Fatal("large icons need roomy rows")
	}
}

func TestIconsOfTheSidebarFollowTheSize(t *testing.T) {
	e := newEnv(t)
	e.m.deps.Cfg.IconSize = "small"
	if _, y := e.find("Home"); strings.Contains(e.sidebarRow(y), "\uf07b") || strings.Contains(e.sidebarRow(y), "\uf015") {
		t.Fatal("small keeps the plain sidebar")
	}
	e.m.deps.Cfg.IconSize = "medium"
	_, y := e.find("Home")
	if !strings.ContainsAny(e.sidebarRow(y), "\uf015\uf07b") {
		t.Fatalf("medium shows an icon before the place: %q", e.sidebarRow(y))
	}
}

// Point 5: fewer or more data per file.
func TestColumnsSettingCyclesBasicFullAndCustom(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	e.key(",")
	e.pickSetting("Columns")
	e.mustHave("‹ custom ›")
	e.key("right")
	e.mustHave("‹ basic ›")
	if got := e.savedConfig(path).Columns; len(got) != 4 {
		t.Fatalf("basic saves four columns: %v", got)
	}
	e.key("right")
	e.mustHave("‹ full ›")
	if got := e.savedConfig(path).Columns; len(got) != len(config.ColumnPresets["full"]) {
		t.Fatalf("full saves every column: %v", got)
	}
}

func TestHeaderMenuOffersBasicAndFullAndTheNewColumns(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	x, y := e.find("Kind")
	e.rightClick(x, y)
	for _, want := range []string{"Basic", "Full", "Owner", "Path"} {
		e.mustHave(want)
	}
	e.clickText("Full")
	if got := e.savedConfig(path).Columns; len(got) != len(config.ColumnPresets["full"]) {
		t.Fatalf("saved %v", got)
	}
	e.send(tea.WindowSizeMsg{Width: 240, Height: 40})
	e.mustHave("Owner")
	e.mustHave("Permissions")
}

func TestNarrowWindowDropsWholeColumnsNeverCutsText(t *testing.T) {
	e := newEnv(t)
	e.m.deps.Cfg.Columns = config.ColumnPresets["full"]
	for _, w := range []int{80, 100, 120} {
		e.send(tea.WindowSizeMsg{Width: w, Height: 40})
		for _, s := range e.m.columns() {
			if strings.Contains(e.listCells(headerRow), "…") {
				t.Fatalf("width %d cuts the header of %s", w, s.col)
			}
		}
	}
}

func TestPlainTextTakesTheThemeTextColorOnALightTheme(t *testing.T) {
	e := newEnv(t)
	e.m.deps.Cfg.Theme = "catppuccin-latte"
	e.m.restyle()
	if !strings.Contains(rawLine(e, "notes.txt"), "\x1b[38;2;76;79;105m") { // latte text #4c4f69
		t.Fatalf("a row of a light theme must set its own text color: %q", rawLine(e, "notes.txt"))
	}
}
