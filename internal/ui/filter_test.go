package ui

import "testing"

func typeText(e *env, s string) {
	for _, r := range s {
		e.key(string(r))
	}
}

func TestQuickFilterNarrowsTheFolderWhileTyping(t *testing.T) {
	e := newEnv(t)
	e.mustHave("data.xlsx")
	e.key("f")
	e.mustHave("Filter:")
	typeText(e, "NOT")
	e.mustHave("notes.txt")
	e.mustNotHave("data.xlsx")
	e.mustNotHave("alpha")
}

func TestQuickFilterAcceptsAGlob(t *testing.T) {
	e := newEnv(t)
	e.key("f")
	typeText(e, "*.xlsx")
	e.mustHave("data.xlsx")
	e.mustNotHave("notes.txt")
}

func TestEnterKeepsTheFilterAndEscClearsIt(t *testing.T) {
	e := newEnv(t)
	e.key("f")
	typeText(e, "notes")
	e.key("enter")
	e.mustNotHave("Filter:")    // the prompt is gone…
	e.mustHave("filter: notes") // …but the toolbar says a filter is on
	e.mustNotHave("data.xlsx")
	e.key("j") // the list works as usual on the filtered rows
	e.key("esc")
	e.mustHave("data.xlsx")
	e.mustNotHave("filter: notes")
}

func TestEscInsideThePromptClearsTheFilter(t *testing.T) {
	e := newEnv(t)
	e.key("f")
	typeText(e, "notes")
	e.key("esc")
	e.mustHave("data.xlsx")
}

func TestTheFilterEndsWhenYouLeaveTheFolder(t *testing.T) {
	e := newEnv(t)
	e.key("f")
	typeText(e, "alpha")
	e.key("enter")
	e.key("enter") // into alpha
	e.mustHave("inner.txt")
	e.mustNotHave("filter: alpha")
}
