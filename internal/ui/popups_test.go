package ui

import (
	"strings"
	"testing"
)

func TestPopupsShowTheEscHintInTheirBottomBorderLikeLazymark(t *testing.T) {
	e := newEnv(t)
	e.key("?")
	if !strings.Contains(e.text(), "[Esc]─╯") {
		t.Fatalf("the help popup has no [Esc] hint in its bottom border:\n%s", e.text())
	}
	e.key("esc")
	e.key(",")
	if !strings.Contains(e.text(), "[Esc]─╯") {
		t.Fatalf("the settings popup has no [Esc] hint:\n%s", e.text())
	}
}

func TestAutoLanguageShowsWhatItResolvedTo(t *testing.T) {
	e := newEnv(t)
	e.key(",")
	if !strings.Contains(e.text(), "auto · ") {
		t.Fatalf("the Language row must say what auto resolved to (like lazymark):\n%s", e.text())
	}
}
