package ui

import (
	"strings"
	"testing"
)

func TestNewTabStartsInTheSameFolderAndShowsTheCount(t *testing.T) {
	e := newEnv(t)
	if strings.Contains(e.text(), "Tab 1/") {
		t.Fatal("a single tab shows no tab label")
	}
	e.key("ctrl+t")
	e.mustHave("Tab 2/2")
	e.mustHave("proj")
}

func TestTabsKeepTheirOwnFolderAndHistory(t *testing.T) {
	e := newEnv(t)
	e.key("ctrl+t")
	e.clickText("alpha")
	e.key("enter") // into alpha, in tab 2
	e.mustHave("inner.txt")
	e.key("{")
	e.mustHave("Tab 1/2")
	e.mustNotHave("inner.txt")
	e.mustHave("beta")
	e.key("}")
	e.mustHave("inner.txt")
	e.key("{")
	if e.m.hist.CanBack() {
		t.Fatal("tab 1 never navigated, so it has no history")
	}
}

func TestTabsWrapAround(t *testing.T) {
	e := newEnv(t)
	e.key("ctrl+t")
	e.key("}")
	e.mustHave("Tab 1/2")
	e.key("{")
	e.mustHave("Tab 2/2")
}

func TestClosingATabReturnsToANeighbourAndTheLastOneStays(t *testing.T) {
	e := newEnv(t)
	e.key("ctrl+t")
	e.clickText("alpha")
	e.key("enter")
	e.key("ctrl+w")
	e.mustNotHave("/2")
	e.mustHave("beta") // back in the folder of tab 1
	e.key("ctrl+w")
	e.mustHave("last tab")
	e.mustHave("beta")
}

func TestTabsKeepTheirOwnExpandedFolders(t *testing.T) {
	e := newEnv(t)
	e.clickText("alpha")
	e.key("+")
	e.mustHave("inner.txt")
	e.key("ctrl+t") // a new tab starts as a copy
	e.mustHave("inner.txt")
	e.key("-")
	e.mustNotHave("inner.txt")
	e.key("{")
	e.mustHave("inner.txt")
}
