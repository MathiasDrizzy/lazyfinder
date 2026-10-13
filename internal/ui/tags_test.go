package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func tagSelected(e *env, name string) {
	e.key("t")
	typeText(e, name)
	e.key("enter")
}

func TestTagToggleOnTheSelectionAndSearchByIt(t *testing.T) {
	e := newEnv(t)
	e.clickText("notes.txt")
	tagSelected(e, "work")
	e.mustHave("Tagged")
	if got := e.m.tags[filepath.Join(e.root, "notes.txt")]; len(got) != 1 || got[0] != "work" {
		t.Fatalf("tags = %v", e.m.tags)
	}
	e.key("/")
	typeText(e, "tag:work")
	e.mustHave("notes.txt")
	e.mustNotHave("data.xlsx")
}

func TestTaggingAgainRemovesTheTag(t *testing.T) {
	e := newEnv(t)
	e.clickText("notes.txt")
	tagSelected(e, "work")
	tagSelected(e, "work")
	if len(e.m.tags) != 0 {
		t.Fatalf("tags = %v", e.m.tags)
	}
	e.mustHave("Untagged")
}

func TestTagActsOnAllMarkedRows(t *testing.T) {
	e := newEnv(t)
	e.clickText("data.xlsx")
	e.key("v")
	e.key("v") // data.xlsx and notes.txt
	tagSelected(e, "both")
	if len(e.m.tags) != 2 {
		t.Fatalf("tags = %v", e.m.tags)
	}
}

func TestEmptyTagNameIsIgnoredAndInfoListsTags(t *testing.T) {
	e := newEnv(t)
	e.clickText("notes.txt")
	tagSelected(e, "  ")
	if len(e.m.tags) != 0 {
		t.Fatalf("tags = %v", e.m.tags)
	}
	tagSelected(e, "red")
	e.key("i")
	e.mustHave("Tags")
	e.mustHave("red")
}

func TestTagsFollowARenameAndPersist(t *testing.T) {
	e := newEnv(t)
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	d := e.m.deps
	d.ConfigPath = cfgPath
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.clickText("notes.txt")
	tagSelected(e, "work")
	e.key("f2")
	e.key("ctrl+u")
	typeText(e, "renamed.txt")
	e.key("enter")
	if _, ok := e.m.tags[filepath.Join(e.root, "renamed.txt")]; !ok {
		t.Fatalf("the tag must follow the rename: %v", e.m.tags)
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(cfgPath), "tags.json"))
	if err != nil || !strings.Contains(string(data), "renamed.txt") {
		t.Fatalf("tags.json: %v %s", err, data)
	}
}

func TestTagPickerSearchesTheChosenTag(t *testing.T) {
	e := newEnv(t)
	e.clickText("notes.txt")
	tagSelected(e, "work")
	e.key("T")
	e.mustHave("Tags")
	e.key("enter")
	if e.m.search.input != "tag:work" {
		t.Fatalf("search = %q", e.m.search.input)
	}
	e.mustHave("notes.txt")
}
