package ui

import (
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/git"
	"github.com/charmbracelet/x/ansi"
)

// Text from a repository or the disk that is not a file name went through showControls, which keeps a color
// sequence: a diff line with ESC [8m hid itself, or a - line painted as a + line, and the same in a stash message, a
// changed path, a link target and a popup title, where an OSC in a file name also cut the frame. Each shows its
// control characters as pictures.
func TestTextFromOutsideShowsItsSequences(t *testing.T) {
	e := newEnv(t)
	const hidden = "x\x1b[8mhidden"
	const want = "x␛[8mhidden"
	shows := func(what string, lines []string) {
		t.Helper()
		if got := ansi.Strip(strings.Join(lines, "\n")); !strings.Contains(got, want) {
			t.Errorf("%s hides the sequence:\n%s", what, got)
		}
	}
	shows("a popup title", e.m.box(hidden, []string{"body"}, 30))
	e.m.diff = diffState{open: true, title: "d.txt", lines: []string{"+" + hidden}}
	_, _, lines := e.m.diffBox()
	shows("a diff line", lines)
	e.m.info.d.Link = hidden
	shows("a link target", e.m.infoLines())
	shows("a stash message", []string{e.m.scmRowLine(0, scmRow{kind: scmStash, ch: git.Change{Path: hidden}})})
	shows("a changed path", []string{e.m.scmRowLine(0, scmRow{kind: scmFile, ch: git.Change{Path: hidden, Code: git.Modified}})})
	if top := ansi.Strip(e.m.box("evil"+oscTitle+".txt", []string{"body"}, 30)[0]); !strings.HasSuffix(top, "╮") {
		t.Errorf("an OSC in a title cut the frame: %q", top)
	}
}
