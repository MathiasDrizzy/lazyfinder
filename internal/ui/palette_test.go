package ui

import (
	"strings"
	"testing"
)

// : opens the command palette: typing narrows the actions by a fuzzy match, Enter runs the one selected. Letters
// type into it, so q does not quit and j does not move.
func TestThePaletteFindsAndRunsAnAction(t *testing.T) {
	e := newEnv(t)
	e.key(":")
	if !e.m.picker.open || len(e.m.picker.items) < 40 {
		t.Fatalf("palette: open %v with %d actions", e.m.picker.open, len(e.m.picker.items))
	}
	quit := e.m.onKey(keyMsg("q"))
	e.typeText("j")
	if quit != nil || e.m.picker.query != "qj" {
		t.Fatalf("q and j must type: q gave a command %v, query %q", quit != nil, e.m.picker.query)
	}
	e.key("backspace")
	e.key("backspace")
	e.typeText("hdn") // fuzzy: Hidden
	if len(e.m.picker.items) == 0 || !strings.Contains(strings.ToLower(e.m.picker.items[0].label), "hidden") {
		t.Fatalf("hdn matched %+v", e.m.picker.items)
	}
	before := e.m.hidden
	e.key("enter")
	if e.m.picker.open || e.m.hidden == before {
		t.Fatalf("enter: open %v, hidden %v -> %v", e.m.picker.open, before, e.m.hidden)
	}
}

// ctrl+p opens the palette too; a query that matches nothing shows an empty list, and Enter then does nothing.
func TestThePaletteWithNoMatch(t *testing.T) {
	e := newEnv(t)
	e.key("ctrl+p")
	e.typeText("zzzzzz")
	if !e.m.picker.open || len(e.m.picker.items) != 0 {
		t.Fatalf("open %v, items %d", e.m.picker.open, len(e.m.picker.items))
	}
	e.m.View() // drawing an empty palette must not fail
	e.key("enter")
	e.key("esc")
	if e.m.picker.open {
		t.Fatal("esc must close the palette")
	}
}

// A whole word beats letters spread apart, and items that tie keep their order.
func TestFuzzyFilterRanksWholeWordsFirst(t *testing.T) {
	items := []pickItem{{label: "Rename"}, {label: "Reverse the sort"}, {label: "Reload"}, {label: "Open", match: "open"}}
	got := fuzzyFilter(items, "re")
	if len(got) != 3 || got[0].label != "Rename" || got[1].label != "Reverse the sort" || got[2].label != "Reload" {
		t.Fatalf("re: %+v", got)
	}
	got = fuzzyFilter(items, "rld")
	if len(got) != 1 || got[0].label != "Reload" {
		t.Fatalf("rld: %+v", got)
	}
	if got := fuzzyFilter(items, "  "); len(got) != len(items) {
		t.Fatalf("blank query must keep every item, got %d", len(got))
	}
	if a, b := fuzzyScore("a-b", "ab"), fuzzyScore("a---b", "ab"); a < 1 || a >= b || fuzzyScore("ba", "ab") != -1 {
		t.Fatalf("letters closer together must score better: %d, %d", a, b)
	}
}

// A click on an item of the palette runs that item, below the row of the query.
func TestAClickInThePaletteRunsTheItem(t *testing.T) {
	e := newEnv(t)
	e.key(":")
	e.typeText("hidden")
	before := e.m.hidden
	x, y := e.find(e.m.picker.items[0].label)
	e.click(x, y)
	if e.m.picker.open || e.m.hidden == before {
		t.Fatalf("click: open %v, hidden %v -> %v", e.m.picker.open, before, e.m.hidden)
	}
}
