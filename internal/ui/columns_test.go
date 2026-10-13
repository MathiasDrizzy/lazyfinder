package ui

import (
	tea "charm.land/bubbletea/v2"
	"regexp"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

func TestSpanishTitlesAreNotCut(t *testing.T) {
	t.Cleanup(func() { i18n.SetLanguage("en") })
	i18n.SetLanguage("es")
	e := newEnv(t)
	e.mustHave("Fecha de modificación")
	e.mustHave("Hoja de cálculo de Excel")
	if strings.Contains(e.text(), "…") {
		t.Fatalf("a column is still cut:\n%s", e.text())
	}
}

func TestExtraColumnsShowExtensionPermissionsAndLastOpened(t *testing.T) {
	e := newEnv(t)
	e.m.deps.Cfg.Columns = []string{"name", "opened", "extension", "permissions", "size"}
	e.send(sizeMsg())
	for _, want := range []string{"Date Last Opened", "Extension", "Permissions", "xlsx", "-rw-r--r--"} {
		e.mustHave(want)
	}
}

func TestDateFormatNumericWritesYearMonthDay(t *testing.T) {
	e := newEnv(t)
	e.m.deps.Cfg.DateFormat = "numeric"
	e.send(sizeMsg())
	if !regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}`).MatchString(e.text()) {
		t.Fatalf("no numeric date:\n%s", e.text())
	}
}

func TestRightClickOnTheHeaderChoosesColumns(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	e.send(tea.WindowSizeMsg{Width: 170, Height: 40})
	x, y := e.find("Date Created")
	e.rightClick(x, y)
	e.mustHave("✓ Date Created")
	e.mustHave("Extension")
	e.clickText("Extension")
	if cols := e.m.deps.Cfg.Columns; cols[len(cols)-1] != "extension" {
		t.Fatalf("extension is added at the end: %v", cols)
	}
	if got := e.savedConfig(path).Columns; got[len(got)-1] != "extension" {
		t.Fatalf("saved %v", got)
	}
	x, y = e.find("Date Created")
	e.rightClick(x, y)
	e.clickText("✓ Date Created") // hides it
	if e.has("Date Created") {
		t.Fatal("the column should be gone")
	}
	for _, c := range e.m.deps.Cfg.Columns {
		if c == "created" {
			t.Fatal("still in config")
		}
	}
}

func TestRightClickOnTheHeaderMovesAColumn(t *testing.T) {
	e := newEnv(t)
	e.withConfigFile()
	x, y := e.find("Size")
	e.rightClick(x, y)
	e.clickText("Move left")
	cols := e.m.deps.Cfg.Columns
	if cols[2] != "size" {
		t.Fatalf("size moved left: %v", cols)
	}
}

func TestComfortableDensityDoublesTheRowHeight(t *testing.T) {
	e := newEnv(t)
	e.m.deps.Cfg.Density = "comfortable"
	e.send(sizeMsg())
	_, ya := e.find("alpha")
	_, yb := e.find("beta")
	if yb-ya != 2 {
		t.Fatalf("rows are %d lines apart", yb-ya)
	}
	e.click(e.m.listX()+8, yb) // a click on the second row selects it
	if e, _ := e.m.selected(); e.Name != "beta" {
		t.Fatalf("selected %q", e.Name)
	}
	for range 4 {
		e.key("down")
	}
	if len(e.m.entries()) < 6 {
		t.Skip("fixture too small")
	}
	e.mustHave("notes.txt") // scrolling keeps the selection visible
}

func TestSettingsHaveDateFormatAndDensityRows(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	e.m.set = settingsState{open: true, sel: setDateFormat}
	e.mustHave("Date format")
	e.key("right")
	if e.m.deps.Cfg.DateFormat != "relative" {
		t.Fatalf("date format %q", e.m.deps.Cfg.DateFormat)
	}
	e.m.set.sel = setDensity
	e.key("right")
	if e.m.deps.Cfg.Density != "normal" {
		t.Fatalf("density %q, want normal after compact", e.m.deps.Cfg.Density)
	}
	e.key("right")
	if e.m.deps.Cfg.Density != "comfortable" {
		t.Fatalf("density %q", e.m.deps.Cfg.Density)
	}
	cfg := e.savedConfig(path)
	if cfg.DateFormat != "relative" || cfg.Density != "comfortable" {
		t.Fatalf("saved %q %q", cfg.DateFormat, cfg.Density)
	}
	_ = fsmodel.ColName
}
