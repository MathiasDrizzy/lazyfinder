package ui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
)

// withIndex gives the model a fake system index that knows one file outside the folder.
func (e *env) withIndex(paths ...string) *[]string {
	asked := &[]string{}
	d := e.m.deps
	d.Index = func(_ context.Context, text string, content bool) ([]string, error) {
		kind := "name"
		if content {
			kind = "content"
		}
		*asked = append(*asked, kind+":"+text)
		return paths, nil
	}
	e.m = New(d)
	e.send(sizeMsg())
	return asked
}

func TestToolbarShowsASearchFieldAndAClickOpensIt(t *testing.T) {
	e := newEnv(t)
	if r := e.screen()[0]; !strings.Contains(r, "Search") {
		t.Fatalf("the top row needs a Search field: %q", r)
	}
	e.clickText("Search")
	if e.m.search.mode != searchTyping {
		t.Fatal("clicking the field must focus it")
	}
	e.typeText("notes")
	if r := e.screen()[0]; !strings.Contains(r, "notes") {
		t.Fatalf("the field shows what is typed: %q", r)
	}
}

func TestTabCyclesTheScopeThroughTheSystemIndex(t *testing.T) {
	e := newEnv(t)
	outside := filepath.Join(e.home, "elsewhere")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	far := filepath.Join(outside, "report-far.txt")
	if err := os.WriteFile(far, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	asked := e.withIndex(far)
	e.key("/")
	e.typeText("report")
	e.mustNotHave("report-far.txt")
	e.key("tab") // subfolders
	e.mustHave("Subfolders")
	e.key("tab") // the whole system
	e.mustHave("System")
	e.mustHave("report-far.txt")
	if len(*asked) == 0 || (*asked)[len(*asked)-1] != "name:report" {
		t.Fatalf("the index was asked %v", *asked)
	}
	e.key("tab")
	e.mustHave("This folder")
}

func TestKindsKeyNarrowsToFilesOrFolders(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.typeText("a")
	e.mustHave("alpha")
	e.mustHave("data.xlsx")
	e.key("ctrl+k") // files
	e.mustNotHave("alpha")
	e.mustHave("data.xlsx")
	e.key("ctrl+k") // folders
	e.mustHave("alpha")
	e.mustNotHave("data.xlsx")
	e.key("ctrl+k") // both
	e.mustHave("data.xlsx")
}

func TestByContentSearchesInsideTheFiles(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.typeText("second")
	e.mustNotHave("notes.txt")
	e.key("ctrl+n") // by content
	e.mustHave("notes.txt")
	e.mustNotHave("data.xlsx")
}

func TestSearchChoicesAreRememberedInTheConfig(t *testing.T) {
	e := newEnv(t)
	d := e.m.deps
	d.ConfigPath = filepath.Join(e.home, "config.json")
	e.m = New(d)
	e.send(sizeMsg())
	e.key("/")
	e.key("tab")
	e.key("ctrl+k")
	e.key("ctrl+n")
	raw, err := os.ReadFile(d.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Parse(raw, e.home)
	if err != nil {
		t.Fatal(err)
	}
	if s := cfg.Search; s.Scope != "subfolders" || s.Kinds != "files" || s.By != "content" {
		t.Fatalf("saved %+v", s)
	}
	e2 := New(Deps{Home: e.home, Start: e.root, Cfg: cfg, Runner: e.run, Now: d.Now, Wake: noWake})
	e2.Update(sizeMsg())
	e2.openSearch()
	if e2.search.scope != "subfolders" || e2.search.kinds != "files" || e2.search.by != "content" {
		t.Fatalf("a new search starts with %+v", e2.search)
	}
}

func TestSettingsHasTheThreeSearchRows(t *testing.T) {
	e := newEnv(t)
	if e.m.settingValue(setSearchScope) != "This folder" || e.m.settingValue(setSearchKinds) != "Files and folders" || e.m.settingValue(setSearchBy) != "Name" {
		t.Fatalf("values %q %q %q", e.m.settingValue(setSearchScope), e.m.settingValue(setSearchKinds), e.m.settingValue(setSearchBy))
	}
	e.m.change(setSearchScope, 1)
	if e.m.deps.Cfg.Search.Scope != "subfolders" {
		t.Fatalf("scope = %q", e.m.deps.Cfg.Search.Scope)
	}
}

func TestRecentsIsAPlaceOfTheSidebar(t *testing.T) {
	e := newEnv(t)
	e.mustHave("Recents")
	e.m.noteVisit(filepath.Join(e.root, "alpha"))
	e.clickText("Recents")
	if !e.m.picker.open {
		t.Fatal("clicking Recents lists the recent folders")
	}
}

func sizeMsg() tea.WindowSizeMsg { return tea.WindowSizeMsg{Width: 120, Height: 40} }
