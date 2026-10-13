package ui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestRecentsListVisitedFoldersNewestFirstAndOpenOne(t *testing.T) {
	e := newEnv(t)
	e.clickText("alpha")
	e.key("enter")
	e.key("backspace")
	e.clickText("beta")
	e.key("enter") // now in beta; recents: beta, proj, alpha
	e.key("E")
	e.mustHave("Recent files and folders")
	if got := e.m.picker.items[0].label; got != "beta" {
		t.Fatalf("first = %q, want the newest folder", got)
	}
	e.key("down")
	e.key("enter")
	if filepath.Base(e.m.dir()) != "proj" {
		t.Fatalf("enter must open the chosen folder, in %s", e.m.dir())
	}
	e.mustNotHave("Recent files and folders")
}

func TestRecentsAreCappedAndDeduplicated(t *testing.T) {
	e := newEnv(t)
	for i := 0; i < 3; i++ {
		e.clickText("alpha")
		e.key("enter")
		e.key("backspace")
	}
	n := 0
	for _, p := range e.m.recents {
		if filepath.Base(p) == "alpha" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("alpha appears %d times in %v", n, e.m.recents)
	}
}

func TestRecentsSurviveARestart(t *testing.T) {
	e := newEnv(t)
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	d := e.m.deps
	d.ConfigPath = cfgPath
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.clickText("alpha")
	e.key("enter")
	if _, err := os.Stat(filepath.Join(filepath.Dir(cfgPath), "recents.json")); err != nil {
		t.Fatal("recents.json must be written next to config.json:", err)
	}
	again := New(d)
	found := false
	for _, p := range again.recents {
		found = found || p == filepath.Join(e.root, "alpha")
	}
	if !found {
		t.Fatalf("recents after restart = %v", again.recents)
	}
}

func TestRecentsDropFoldersThatNoLongerExist(t *testing.T) {
	e := newEnv(t)
	e.m.recents = []string{filepath.Join(e.root, "gone"), filepath.Join(e.root, "beta")}
	e.key("E")
	e.mustNotHave("gone")
	e.mustHave("beta")
}

func TestRecentsAlsoListTheFilesYouOpenAndRevealThem(t *testing.T) {
	e := newEnv(t)
	e.clickText("notes.txt")
	e.key("enter") // opens the file with the configured program
	e.clickText("alpha")
	e.key("enter")
	e.key("E")
	var labels []string
	for _, it := range e.m.picker.items {
		labels = append(labels, it.label)
	}
	if !slices.Contains(labels, "notes.txt") {
		t.Fatalf("the opened file is not in the recents: %v", labels)
	}
	for i, it := range e.m.picker.items {
		if it.label == "notes.txt" {
			e.m.picker.sel = i
		}
	}
	e.key("enter")
	if e.m.dir() != e.root {
		t.Fatalf("choosing a file shows its folder, in %s", e.m.dir())
	}
	if en, ok := e.m.selected(); !ok || en.Name != "notes.txt" {
		t.Fatalf("the file should be selected, got %+v", en)
	}
}
