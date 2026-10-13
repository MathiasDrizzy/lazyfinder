package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The filters sit on their own row above the prompt, as labelled buttons (never in brackets).
func TestSearchFiltersAreLabelledButtonsOnTheirOwnRow(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	lines := e.screen()
	filters, prompt := lines[len(lines)-2], lines[len(lines)-1]
	for _, want := range []string{"Look in (Tab):", "This folder ▾", "What (^K):", "Files and folders ▾", "By (^N):", "Name ▾"} {
		if !strings.Contains(filters, want) {
			t.Fatalf("filter row = %q, want %q", filters, want)
		}
	}
	if strings.ContainsAny(filters, "[]") {
		t.Fatalf("filter row = %q, buttons must not use brackets", filters)
	}
	if strings.Contains(prompt, "[") || strings.Contains(prompt, "This folder") {
		t.Fatalf("prompt row = %q, must hold only the prompt and the count", prompt)
	}
	if strings.Contains(prompt, "kind:") {
		t.Fatalf("prompt row = %q, the operators are explained in the help", prompt)
	}
}

func TestChangingAFilterLeavesNoLooseText(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.key("tab")
	lines := e.screen()
	filters, prompt := lines[len(lines)-2], lines[len(lines)-1]
	if !strings.Contains(filters, "Subfolders ▾") {
		t.Fatalf("filter row = %q, want the new value", filters)
	}
	if strings.Contains(prompt, "Subfolders") || strings.Contains(prompt, "Files and folders") {
		t.Fatalf("prompt row = %q, the value must show only in its button", prompt)
	}
}

func TestSearchFiltersNeverGetCut(t *testing.T) {
	for _, w := range []int{50, 60, 70, 80, 100, 120} {
		e := newEnv(t)
		e.send(tea.WindowSizeMsg{Width: w, Height: 30})
		e.key("/")
		lines := e.screen()
		filters := lines[len(lines)-2]
		if strings.Contains(filters, "…") {
			t.Fatalf("width %d: filter row = %q, nothing may be cut", w, filters)
		}
		for _, label := range []string{"Images", "Text", "Folders", "Today", "Week"} {
			if strings.Contains(filters, label[:2]) != strings.Contains(filters, label) {
				t.Fatalf("width %d: %q is cut in %q", w, label, filters)
			}
		}
	}
}

func TestEachSearchFilterButtonCanBeClicked(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.clickText("This folder ▾")
	if e.m.search.scope != "subfolders" {
		t.Fatalf("scope = %q, want subfolders", e.m.search.scope)
	}
	e.clickText("Files and folders ▾")
	if e.m.search.kinds != "files" {
		t.Fatalf("kinds = %q, want files", e.m.search.kinds)
	}
	e.clickText("Name ▾")
	if e.m.search.by != "content" {
		t.Fatalf("by = %q, want content", e.m.search.by)
	}
}

func TestClickingThePromptRowReturnsTheKeyboardToTheField(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.typeText("notes")
	e.key("enter")
	if e.m.search.mode != searchResults {
		t.Fatal("enter must hand the keyboard to the results")
	}
	e.click(5, e.m.h-1)
	if e.m.search.mode != searchTyping {
		t.Fatal("a click on the field must give it the keyboard")
	}
}

func TestHelpExplainsTheSearchOperators(t *testing.T) {
	e := newEnv(t)
	e.key("?")
	e.mustHave("kind:image")
}
