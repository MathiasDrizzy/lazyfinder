package ui

import (
	"strings"
	"testing"
)

// Get Info walked a folder inside Update, so on / or a home folder the program froze until the walk ended. It now
// opens at once, counts apart, and a result that arrives after the popup closed is dropped.
func TestGetInfoOfAFolderCountsApartFromTheUI(t *testing.T) {
	e := newEnv(t)
	e.pick("alpha")
	count := e.m.openInfo()
	if count == nil || !strings.Contains(strings.Join(e.m.infoLines(), "\n"), "Counting") {
		t.Fatalf("the popup did not open counting apart: %q", e.m.infoLines())
	}
	e.send(count())
	e.mustHave("2 files")

	again := e.m.openInfo()
	e.key("esc")
	e.send(again())
	if e.m.info.open {
		t.Fatal("a count that arrived after Esc opened the popup again")
	}
}
