package ui

import (
	"fmt"
	"slices"
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

// Order 2026-10-13-06, point 1 (as corrected by Mathias): Favorites and Locations are two boxes, each with its own
// border and title, like Folders, Tasks and Tags in lazymark; no "—" line between them.
func TestSidebarIsTwoBoxesFavoritesAndLocations(t *testing.T) {
	e := newEnv(t)
	w := e.m.sidebarW()
	var tops, bottoms []int
	for y := 1; y < e.m.panelH()+1; y++ {
		row := string(e.cells(y)[:w])
		if strings.HasPrefix(row, "╭") {
			tops = append(tops, y)
		}
		if strings.HasPrefix(row, "╰") {
			bottoms = append(bottoms, y)
		}
		if strings.Trim(strings.TrimSpace(e.sidebarRow(y)), "─") == "" && strings.Contains(e.sidebarRow(y), "──") && !strings.HasPrefix(row, "╭") && !strings.HasPrefix(row, "╰") {
			t.Fatalf("row %d is a divider line, not a box:\n%s", y, e.text())
		}
	}
	if len(tops) != 2 || len(bottoms) != 2 {
		t.Fatalf("want two boxes, got tops %v bottoms %v:\n%s", tops, bottoms, e.text())
	}
	if !strings.Contains(string(e.cells(tops[0])[:w]), "Favorites") || !strings.Contains(string(e.cells(tops[1])[:w]), "Locations") {
		t.Fatalf("each box carries its title:\n%s", e.text())
	}
	if e.has("Places") {
		t.Fatal("the old single Places box is gone")
	}
	if bottoms[0] >= tops[1] {
		t.Fatalf("the Favorites box must end before the Locations box starts: %v %v", bottoms, tops)
	}
}

func TestBoxBordersOfTheSidebarAreNotClickable(t *testing.T) {
	e := newEnv(t)
	before := e.m.dir()
	_, y := e.find("Locations")
	e.click(3, y) // the title edge of the Locations box
	e.click(3, y-1)
	if e.m.dir() != before {
		t.Fatalf("clicking a box edge moved to %s", e.m.dir())
	}
}

func TestClickingAnItemInEachBoxOpensIt(t *testing.T) {
	e := newEnv(t)
	for _, name := range []string{"Home", "Disk"} {
		x, y := e.find(name)
		e.click(x, y)
		if e.m.sideSel < 0 {
			t.Fatal("selection lost")
		}
		if p, ok := e.m.selectedPlace(); !ok || p.name != name {
			t.Fatalf("clicking %s selected %+v", name, p)
		}
	}
}

func TestTheSidebarBoxesFitAShortWindowWithoutBreaking(t *testing.T) {
	e := newEnv(t)
	e.send(tea.WindowSizeMsg{Width: 100, Height: 16})
	w := e.m.sidebarW()
	for y := 1; y <= e.m.panelH(); y++ {
		if got := len([]rune(string(e.cells(y)[:w]))); got != w {
			t.Fatalf("row %d is %d cells wide, want %d", y, got, w)
		}
	}
	if !strings.Contains(e.text(), "Favorites") || !strings.Contains(e.text(), "Locations") {
		t.Fatalf("both boxes must stay at the minimum size:\n%s", e.text())
	}
}

// Point 4 (as corrected by Mathias): a │ between the column names in the header only, and dragging it still resizes.
func (e *env) listCells(y int) string {
	r := e.cells(y)
	return string(r[e.m.listX() : e.m.listX()+e.m.listW()])
}

func TestColumnsAreSeparatedByABarLikeFinder(t *testing.T) {
	e := newEnv(t)
	n := len(e.m.columns()) - 1
	if got := strings.Count(e.listCells(headerRow), "│"); got != n {
		t.Fatalf("the header has %d separators, want %d:\n%s", got, n, e.text())
	}
	// Mathias: "no quería que los | fueran hasta abajo": only the names of the columns carry the bar.
	for y := firstRow; y < e.m.panelH()-1; y++ {
		if strings.Contains(e.listCells(y), "│") {
			t.Fatalf("row %d must not have a column bar:\n%s", y, e.text())
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

// Point 5: fewer or more data per file.
func TestColumnsSettingCyclesBasicFullAndCustom(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	e.key(",")
	e.pickSetting("Columns")
	e.mustHave("‹ basic ›") // the default columns are the basic preset (Mathias, 2026-10-14: the date created in it)
	e.key("right")
	e.mustHave("‹ full ›")
	if got := e.savedConfig(path).Columns; len(got) != len(config.ColumnPresets["full"]) {
		t.Fatalf("full saves every column: %v", got)
	}
	e.key("right")
	e.mustHave("‹ basic ›")
	if got := e.savedConfig(path).Columns; !slices.Contains(got, "created") || len(got) != 5 {
		t.Fatalf("basic saves its five columns, the date created among them: %v", got)
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

// Mathias: the boxes start at half of the sidebar each, and Favorites stretches by itself as folders are added.
func (e *env) boxHeights() (fav, loc int) {
	e.t.Helper()
	l := e.m.sidebarBoxLayout()
	if len(l) != 2 {
		e.t.Fatalf("want two boxes, got %d", len(l))
	}
	return l[0].h, l[1].h
}

func (e *env) withFavorites(n int) {
	favs := make([]config.Place, n)
	for i := range favs {
		favs[i] = config.Place{Name: fmt.Sprintf("fav%02d", i), Path: e.root}
	}
	e.m.setFavorites(favs)
}

func TestTheBoxesFitTheirContentAndLeaveTheRestFree(t *testing.T) {
	e := newEnv(t)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	total := e.m.panelH()
	e.withFavorites(2)
	fav, loc := e.boxHeights()
	l := e.m.sidebarBoxLayout()
	if fav != len(l[0].items)+2 || loc != max(minLocationsH, len(l[1].items)+2) || fav+loc >= total {
		t.Fatalf("2 favorites: boxes %d and %d of %d, want content plus borders and free space left", fav, loc, total)
	}
	e.withFavorites(10)
	if fav2, _ := e.boxHeights(); fav2 != len(e.m.sidebarBoxLayout()[0].items)+2 || fav2 <= fav {
		t.Fatalf("10 favorites: Favorites is %d high, want it to follow its content", fav2)
	}
	e.withFavorites(2)
	if back, _ := e.boxHeights(); back != fav {
		t.Fatalf("removing favorites must give the space back: %d, want %d", back, fav)
	}
}

func TestDraggingTheBorderBetweenTheBoxesResizesFavoritesAndIsRemembered(t *testing.T) {
	e := newEnv(t)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.withFavorites(3)
	fav, _ := e.boxHeights()
	border := 1 + fav - 1 // screen row of the bottom border of Favorites
	e.send(tea.MouseClickMsg{X: 5, Y: border, Button: tea.MouseLeft})
	e.send(tea.MouseMotionMsg{X: 5, Y: border + 4, Button: tea.MouseLeft})
	e.send(tea.MouseReleaseMsg{X: 5, Y: border + 4, Button: tea.MouseLeft})
	if got, _ := e.boxHeights(); got != fav+4 {
		t.Fatalf("Favorites is %d high after the drag, want %d", got, fav+4)
	}
	if want := fav - 2 + 4; e.m.deps.Cfg.FavoritesHeight != want {
		t.Fatalf("favorites_height = %d, want %d", e.m.deps.Cfg.FavoritesHeight, want)
	}
	e.send(tea.MouseClickMsg{X: 5, Y: border + 4, Button: tea.MouseLeft})
	e.send(tea.MouseClickMsg{X: 5, Y: border + 4, Button: tea.MouseLeft})
	if got, _ := e.boxHeights(); got != fav {
		t.Fatalf("a double click on the border fits Favorites to its content: %d, want %d", got, fav)
	}
}

func TestTheBoxesCanBeResizedFromTheKeyboard(t *testing.T) {
	e := newEnv(t)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.withFavorites(3)
	fav, _ := e.boxHeights()
	e.key("ctrl+down")
	if got, _ := e.boxHeights(); got != fav+1 {
		t.Fatalf("ctrl+down: Favorites is %d high, want %d", got, fav+1)
	}
	e.key("ctrl+up")
	e.key("ctrl+up")
	if got, _ := e.boxHeights(); got != fav-1 {
		t.Fatalf("ctrl+up twice: Favorites is %d high, want %d", got, fav-1)
	}
}

func TestLocationsKeepsAVisibleMinimumAndFavoritesScrolls(t *testing.T) {
	e := newEnv(t)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 24})
	e.withFavorites(40)
	fav, loc := e.boxHeights()
	if loc < 4 || fav+loc != e.m.panelH() {
		t.Fatalf("Locations %d must keep at least 4 rows (Favorites %d)", loc, fav)
	}
	e.key("tab")
	for i := 0; i < 30; i++ {
		e.key("down")
	}
	p, _ := e.m.selectedPlace()
	if !e.has(p.name) {
		t.Fatalf("the selected favorite %s must scroll into view:\n%s", p.name, e.text())
	}
	x, y := e.find(p.name)
	e.click(x, y)
	if q, _ := e.m.selectedPlace(); q.name != p.name {
		t.Fatalf("a click on the scrolled box hit %s, want %s", q.name, p.name)
	}
}
