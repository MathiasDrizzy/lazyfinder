package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Trash and Undo ran inside Update: across volumes they copy, and the UI froze without progress, Esc or q until the
// copy ended, and a signal could not be read. They run apart from the UI like a paste.
func TestTrashAndUndoRunApartFromTheUI(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	p := filepath.Join(e.m.dir(), "notes.txt")
	cmd := e.m.trash([]string{p})
	if !e.m.op.busy {
		t.Fatal("the trash ran inside Update")
	}
	e.finish(cmd)
	if _, err := os.Lstat(p); err == nil {
		t.Fatal("the trash did not move the file")
	}
	cmd = e.m.undo()
	if !e.m.op.busy {
		t.Fatal("the undo ran inside Update")
	}
	e.finish(cmd)
	if _, err := os.Lstat(p); err != nil {
		t.Fatalf("the undo did not bring the file back: %v", err)
	}
}

// finish hands the end of an operation started apart from the UI to the model.
func (e *env) finish(cmd tea.Cmd) {
	e.t.Helper()
	for _, msg := range flatten(cmd) {
		if done, ok := msg.(opDoneMsg); ok {
			if done.err != nil {
				e.t.Fatal(done.err)
			}
			e.m.Update(done)
		}
	}
}
