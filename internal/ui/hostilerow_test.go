package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// A name with a sequence in it lost the rest of its row: the BEL that ends an OSC showed as ␇ while its ESC was kept,
// so the frame filter took everything after "evil" (the rest of the name, the date, the size and the kind) for an
// OSC with no end and dropped it. Two such files looked the same. The whole name shows, its ESC as ␛, and the row
// keeps its columns.
func TestAHostileNameKeepsTheRestOfItsRow(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "proj", "alpha")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	names := map[string]string{
		"evil" + oscTitle + "名前.txt":   "PWNED␇名前",
		"csi" + csiClear + "after.txt": "[2Jafter.txt",
		"two" + escReset + "after.txt": "cafter.txt",
	}
	for n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e := newEnvAt(t, home)
	e.send(tea.KeyPressMsg{Code: tea.KeyEnter}) // alpha is the first row
	if e.m.dir() != dir {
		t.Fatalf("did not enter alpha: %s", e.m.dir())
	}
	frame := ansi.Strip(e.m.View().Content)
	for name, rest := range names {
		start := name[:strings.IndexByte(name, '\x1b')]
		var row string
		for _, l := range strings.Split(frame, "\n") {
			if strings.Contains(l, start+"␛") {
				row = l
			}
		}
		if row == "" {
			t.Errorf("%q: no row shows the name with its ESC as ␛:\n%s", name, frame)
			continue
		}
		if !strings.Contains(row, rest) || !strings.Contains(row, "1 byte") {
			t.Errorf("%q: the row lost the rest of the name or its columns: %q", name, row)
		}
	}
}

// A sequence that is a color (SGR) was kept in a name, so "red␛[31mtext" showed as a red "redtext"; a folder whose
// name holds a whole OSC showed as "dirend" in the title and the tab, its OSC dropped by the frame filter; and the
// status line turned control characters into spaces. Each of them names the file as it is, with pictures.
func TestAHostileNameLooksTheSameEverywhere(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "proj", "alpha")
	sub := filepath.Join(dir, "dir"+oscTitle+"end")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "red\x1b[31mtext.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := newEnvAt(t, home)
	e.send(tea.KeyPressMsg{Code: tea.KeyEnter}) // alpha
	if frame := ansi.Strip(e.m.View().Content); !strings.Contains(frame, "red␛[31mtext") {
		t.Errorf("the row hides the color sequence of the name:\n%s", frame)
	}
	if got := e.m.folderLabel(sub); got != "dir␛]0;PWNED␇end" {
		t.Errorf("folder label %q", got)
	}
	e.m.setStatus("Pasted " + filepath.Base(sub) + "\nnext")
	if e.m.status != "Pasted dir␛]0;PWNED␇end next" {
		t.Errorf("status %q", e.m.status)
	}
}
