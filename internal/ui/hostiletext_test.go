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

// shownName ran on a diff line before its tabs were expanded, so indented code showed as ␉␉return and a CRLF file
// ended every line in ␍. A diff keeps its indentation and its CRLF lines read as plain lines.
func TestADiffKeepsItsTabsAndCRLFLines(t *testing.T) {
	e := newEnv(t)
	e.m.diff = diffState{open: true, title: "d.go", lines: diffLines("+\t\treturn x\r\n-old\r\n")}
	_, _, lines := e.m.diffBox()
	got := ansi.Strip(strings.Join(lines, "\n"))
	if strings.ContainsAny(got, "␉␍") || !strings.Contains(got, "+        return x") {
		t.Fatalf("the diff shows its tabs or carriage returns as pictures:\n%s", got)
	}
}

// A name with a right-to-left override, "invoice" U+202E "fdp.exe", can be drawn as invoiceexe.pdf by a terminal
// that applies bidi, so Open and Trash named a program as a PDF. The bidi controls show as �.
func TestABidiOverrideInANameShowsAsAPicture(t *testing.T) {
	for _, r := range []rune{0x202a, 0x202b, 0x202c, 0x202d, 0x202e, 0x2066, 0x2067, 0x2068, 0x2069} {
		if got := shownName("invoice" + string(r) + "fdp.exe"); got != "invoice�fdp.exe" {
			t.Errorf("U+%04X: %q", r, got)
		}
	}
	// The marks reorder nothing and are common in Hebrew and Arabic names: they stay.
	for _, r := range []rune{0x200e, 0x200f, 0x061c} {
		if n := "דוח" + string(r) + ".pdf"; shownName(n) != n {
			t.Errorf("U+%04X was replaced", r)
		}
	}
}
