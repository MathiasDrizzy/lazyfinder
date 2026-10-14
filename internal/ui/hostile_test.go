package ui

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Sequences a file name or a file's text can carry to make the terminal do something: open a link (OSC 8), write
// the clipboard (OSC 52), set the title (OSC 0) or answer a query (DCS). lazyfinder draws none of them itself.
const (
	oscLink      = "\x1b]8;;https:evil\a"
	oscClipboard = "\x1b]52;c;UFdORUQ=\a"
	oscTitle     = "\x1b]0;PWNED\a"
	dcsQuery     = "\x1bP+q544e\x1b\\"
)

// hostileEnv starts in a folder whose names and text carry those sequences and plain control characters.
func hostileEnv(t *testing.T) *env {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, "proj", "alpha")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	names := []string{oscLink + "link.txt", oscClipboard + "clip.txt", oscTitle + "title", "line\nbreak.txt",
		"tab\there.txt", "bell\a.txt", "c1\u009d8;;x\u009c.txt"}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	text := "plain\n" + oscLink + "x\n" + oscClipboard + oscTitle + dcsQuery + "\x1b[2J\rend\n"
	for _, n := range []string{"a-text.txt", "a-note.md"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e := newEnvAt(t, home)
	e.send(tea.KeyPressMsg{Code: tea.KeyEnter}) // alpha is the first row
	if !strings.HasSuffix(e.m.dir(), "alpha") {
		t.Fatalf("did not enter alpha: %s", e.m.dir())
	}
	return e
}

// mustBeSafeFrame fails when the frame carries a string sequence or a control character other than ESC and the line
// breaks, or when a row is wider than the terminal or the frame has more rows than it.
func mustBeSafeFrame(t *testing.T, e *env, where string) {
	t.Helper()
	frame := e.m.View().Content
	for _, intro := range []string{"\x1b]", "\x1bP", "\x1b_", "\x1b^", "\x1bX"} {
		if strings.Contains(frame, intro) {
			t.Errorf("%s: the frame carries the string sequence %q", where, intro)
		}
	}
	for _, r := range frame {
		if (r < 0x20 && r != '\n' && r != '\x1b') || (r >= 0x7f && r < 0xa0) {
			t.Errorf("%s: the frame carries the control character %U", where, r)
			break
		}
	}
	lines := strings.Split(frame, "\n")
	if len(lines) > e.m.h {
		t.Errorf("%s: %d rows for a terminal of %d", where, len(lines), e.m.h)
	}
	for i, l := range lines {
		if w := ansi.StringWidth(l); w > e.m.w {
			t.Errorf("%s: row %d is %d wide in a terminal of %d: %q", where, i, w, e.m.w, ansi.Strip(l))
		}
	}
}

func TestHostileNamesAndTextNeverReachTheTerminal(t *testing.T) {
	e := hostileEnv(t)
	mustBeSafeFrame(t, e, "list")
	e.key(" ")
	mustBeSafeFrame(t, e, "preview of a-note.md")
	e.send(tea.KeyPressMsg{Code: tea.KeyDown})
	mustBeSafeFrame(t, e, "preview of a-text.txt")
	e.send(tea.KeyPressMsg{Code: tea.KeyEscape})
	e.key("M")
	mustBeSafeFrame(t, e, "column view")
	e.key("M")
	e.send(tea.WindowSizeMsg{Width: 40, Height: 30})
	mustBeSafeFrame(t, e, "sidebar mode")
}

func TestFifosAndDevicesNeverBlockTheWindow(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "proj", "alpha")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The first row of each case is the special file, so it is selected as soon as the folder opens.
	if err := syscall.Mkfifo(filepath.Join(dir, "a-pipe.txt"), 0o644); err != nil {
		t.Skip("no FIFOs here:", err)
	}
	for _, link := range []struct{ name, target string }{{"b-tty.txt", "/dev/tty"}, {"c-tty.png", "/dev/tty"}, {"d-zero.txt", "/dev/zero"}} {
		if err := os.Symlink(link.target, filepath.Join(dir, link.name)); err != nil {
			t.Fatal(err)
		}
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		e := newEnvAt(t, home)
		e.send(tea.KeyPressMsg{Code: tea.KeyEnter})
		for range 4 {
			e.key(" ")
			e.send(tea.KeyPressMsg{Code: tea.KeyEscape})
			e.key("M")
			e.key("M")
			e.send(tea.KeyPressMsg{Code: tea.KeyDown})
		}
		e.send(tea.WindowSizeMsg{Width: 40, Height: 30})
		_ = e.m.View()
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("a FIFO or a device in the folder blocked the window")
	}
}
