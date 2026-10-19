package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// A hangup or a kill quit at once, skipping the model, so a copy under way was cut off and left its half copy with no
// step to undo. SignalMsg quits the way q does: the copy is stopped and cleaned up first.
func TestASignalStopsACopyBeforeQuitting(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	dup := e.m.duplicate()
	if _, cmd := e.m.Update(SignalMsg{}); quits(cmd) || !e.m.op.ending {
		t.Fatalf("the signal quit before the copy stopped: ending %v", e.m.op.ending)
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
	if _, cmd := e.m.Update(SignalMsg{}); !quits(cmd) {
		t.Fatal("with nothing under way a signal quits at once")
	}
}

// Bubble Tea answers SIGTERM with a quit that skips the model. While a copy runs, QuitFilter makes it a SignalMsg, so
// the copy is stopped first; the quit the model then asks for, and any quit with nothing under way, go through.
func TestQuitFilterSendsASignalQuitToTheModelWhileACopyRuns(t *testing.T) {
	e := newEnv(t)
	if _, ok := QuitFilter(e.m, tea.QuitMsg{}).(tea.QuitMsg); !ok {
		t.Fatal("with nothing under way the quit must go through")
	}
	e.pick("notes.txt")
	dup := e.m.duplicate()
	msg := QuitFilter(e.m, tea.QuitMsg{})
	if _, ok := msg.(SignalMsg); !ok {
		t.Fatalf("during a copy the quit became %T, want SignalMsg", msg)
	}
	e.m.Update(msg)
	var after tea.Cmd
	for _, m := range flatten(dup) {
		if _, ok := m.(opDoneMsg); ok {
			_, after = e.m.Update(m)
		}
	}
	for _, m := range flatten(after) {
		if _, ok := m.(tea.QuitMsg); ok {
			if _, ok := QuitFilter(e.m, m).(tea.QuitMsg); !ok {
				t.Fatal("the quit the model asked for must go through")
			}
			return
		}
	}
	t.Fatal("the model did not quit once the copy stopped")
}

// SIGINT reaches Bubble Tea's loop as an InterruptMsg, which ended the program skipping the model like SIGTERM did;
// and a signal that came after q, while the copy was being stopped, quit at once and left the half copy.
func TestAnInterruptOrASecondSignalWaitsForTheCopyToStop(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.m.duplicate()
	if _, ok := QuitFilter(e.m, tea.InterruptMsg{}).(SignalMsg); !ok {
		t.Fatal("during a copy an interrupt must reach the model")
	}
	e.m.Update(SignalMsg{})
	if _, cmd := e.m.Update(SignalMsg{}); quits(cmd) {
		t.Fatal("a second signal quit before the copy stopped")
	}
}
