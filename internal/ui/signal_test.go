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
