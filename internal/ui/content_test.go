package ui

import (
	"context"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestContentSearchInTheFolderFindsFilesByTheirText(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	typeText(e, "in:first")
	e.mustHave("notes.txt")
	e.mustNotHave("data.xlsx")
}

func TestContentSearchInSubfoldersFindsDeepFiles(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.key("tab")
	typeText(e, "in:deep")
	e.mustHave("x.txt")
	e.mustNotHave("notes.txt")
}

func TestContentSearchCombinesWithOtherTerms(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.key("tab")
	typeText(e, "in:text ext:txt")
	e.mustHave("inner.txt")
	e.mustNotHave("x.txt")
}

func TestContentSearchUsesTheConfiguredFinder(t *testing.T) {
	e := newEnv(t)
	var asked string
	d := e.m.deps
	d.ContentFind = func(_ context.Context, root, text string, _ bool) ([]string, error) {
		asked = text
		return []string{filepath.Join(root, "data.xlsx")}, nil
	}
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.key("/")
	e.key("tab")
	typeText(e, "in:zzz")
	e.mustHave("data.xlsx")
	if asked != "zzz" {
		t.Fatalf("finder asked for %q", asked)
	}
}
