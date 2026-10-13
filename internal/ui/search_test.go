package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func (e *env) typeText(s string) {
	e.t.Helper()
	for _, r := range s {
		e.send(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func TestSlashOpensTheSearchBarAndTypingFiltersTheFolder(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.mustHave("Search")
	e.typeText("notes")
	e.mustHave("notes.txt")
	e.mustNotHave("data.xlsx")
	e.mustNotHave("alpha")
}

func TestEscClearsTheSearchAndBringsBackTheFolder(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.typeText("notes")
	e.key("esc")
	e.mustHave("data.xlsx")
	e.mustHave("alpha")
	if e.m.search.mode != searchOff {
		t.Fatal("esc must close the search")
	}
}

func TestBackspaceEditsTheQueryAndWidensTheResults(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.typeText("notesx")
	e.mustNotHave("notes.txt")
	e.key("backspace")
	e.mustHave("notes.txt")
}

func TestTabSwitchesToSubfoldersAndShowsWhereEachResultLives(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.typeText("inner")
	e.mustNotHave("inner.txt") // not in this folder
	e.key("tab")
	e.mustHave("inner.txt")
	e.mustHave("alpha")
	e.mustHave("Subfolders")
}

func TestRecursiveSearchFindsDeepFiles(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.key("tab")
	e.typeText("x.txt")
	e.mustHave("x.txt")
	e.mustHave("deep")
}

func TestEnterKeepsTheResultsAndLetsYouUseTheList(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.typeText("txt")
	e.key("enter")
	if e.m.search.mode != searchResults {
		t.Fatalf("mode = %v, want results", e.m.search.mode)
	}
	e.mustNotHave("data.xlsx")
	e.key("down") // navigation keys work again
	e.key("esc")
	e.mustHave("data.xlsx")
}

func TestOpeningAFolderResultLeavesTheSearch(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.key("tab")
	e.typeText("deep")
	e.key("enter")
	e.key("enter") // opens the selected result, the "deep" folder
	if e.m.dir() != filepath.Join(e.root, "alpha", "deep") {
		t.Fatalf("dir = %s", e.m.dir())
	}
	if e.m.search.mode != searchOff {
		t.Fatal("entering a folder ends the search")
	}
}

func TestOpeningAFileResultStartsTheOpener(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.typeText("notes")
	e.key("enter")
	e.key("enter")
	if len(e.run.calls) != 1 || e.run.calls[0][0] != "myeditor" {
		t.Fatalf("calls = %v, want one myeditor", e.run.calls)
	}
}

func TestAKindChipCanBeClicked(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.clickText("Folders")
	if !strings.Contains(e.m.search.input, "kind:folder") {
		t.Fatalf("input = %q, want kind:folder", e.m.search.input)
	}
	e.mustHave("alpha")
	e.mustNotHave("notes.txt")
	e.clickText("Folders") // clicking again removes it
	e.mustHave("notes.txt")
}

func TestTheScopeChipCanBeClicked(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.typeText("inner")
	e.clickText("[This folder]")
	e.mustHave("inner.txt")
}

func TestFilterChipsAndQueryFilterByDateAndSize(t *testing.T) {
	e := newEnv(t)
	old := filepath.Join(e.root, "notes.txt")
	past := e.now.AddDate(-1, 0, 0)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	e.m.reload()
	e.key("/")
	e.typeText("txt modified:today")
	e.mustNotHave("notes.txt")
	e.m.search.input = ""
	e.typeText("size:>1")
	e.mustNotHave("alpha") // folders have no size
}

func TestABadQueryShowsWhyAndKeepsTheFolder(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.typeText("txt size:big")
	e.mustHave("not a size")
	e.send(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	e.typeText("notes")
	e.mustHave("notes.txt") // a valid query again
}

func TestClickingTheSearchGlyphOpensTheBar(t *testing.T) {
	e := newEnv(t)
	e.click(e.m.w-6, 0)
	if e.m.search.mode != searchTyping {
		t.Fatalf("mode = %v, want typing", e.m.search.mode)
	}
}

func TestResultLimitIsReported(t *testing.T) {
	e, _ := liveEnv(t, strings.Replace(cfgJSON, `"version": 1,`, `"version": 1, "search": {"max_results": 2},`, 1))
	e.key("/")
	e.key("tab")
	e.typeText("t")
	e.mustHave("limit reached")
}
