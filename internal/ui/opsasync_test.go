package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Paste copied inside Update: a copy of gigabytes froze the program with no progress and no way to stop it. It now
// runs apart, shows its progress, refuses a second change until it ends, and Esc cancels it.
func TestAPasteRunsApartAndEscCancelsIt(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("c")
	e.pick("alpha")
	e.key("enter")
	paste := e.m.paste()
	if paste == nil || !e.m.op.busy || !strings.Contains(e.m.status, "Copying") {
		t.Fatalf("the paste did not start apart: busy=%v status=%q", e.m.op.busy, e.m.status)
	}
	if e.m.paste() != nil || !strings.Contains(e.m.status, "Wait") {
		t.Fatalf("a second change must wait: %q", e.m.status)
	}
	e.key("esc")
	for _, msg := range flatten(paste) {
		e.send(msg)
	}
	if e.onDisk("alpha/notes.txt") || e.m.op.busy || !strings.Contains(e.m.status, "Cancelled") {
		t.Fatalf("Esc did not cancel the copy: busy=%v status=%q", e.m.op.busy, e.m.status)
	}
	e.key("p")
	if !e.onDisk("alpha/notes.txt") || !strings.Contains(e.m.status, "Pasted") {
		t.Fatalf("the paste after the cancel failed: %q", e.m.status)
	}
}

// Quitting while a copy runs stops it first, so it leaves no half copy, and then quits; a second ctrl+c quits at once.
func TestQuittingDuringACopyStopsItFirst(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("D")
	e.pick("notes.txt")
	dup := e.m.duplicate()
	if _, cmd := e.m.Update(keyMsg("ctrl+c")); quits(cmd) {
		t.Fatal("ctrl+c quit before the copy stopped")
	}
	var after tea.Cmd
	for _, msg := range flatten(dup) {
		if _, ok := msg.(opDoneMsg); ok {
			_, after = e.m.Update(msg)
		}
	}
	if !quits(after) {
		t.Fatal("the program did not quit once the copy stopped")
	}

	e = newEnv(t)
	e.pick("notes.txt")
	_ = e.m.duplicate()
	_, _ = e.m.Update(keyMsg("ctrl+c"))
	if _, cmd := e.m.Update(keyMsg("ctrl+c")); !quits(cmd) {
		t.Fatal("a second ctrl+c must quit at once")
	}
}

func quits(cmd tea.Cmd) bool {
	for _, msg := range flatten(cmd) {
		if _, ok := msg.(tea.QuitMsg); ok {
			return true
		}
	}
	return false
}

// Once Esc asked a copy to stop, the next progress tick wrote "Copying… (Esc cancels)" over "Cancelling…".
func TestCancellingStaysOnTheStatusLine(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	_ = e.m.duplicate()
	_, _ = e.m.Update(keyMsg("esc"))
	_, _ = e.m.Update(opTickMsg{})
	if !strings.Contains(e.m.status, "Cancelling") {
		t.Fatalf("status = %q after a tick, want Cancelling…", e.m.status)
	}
}
