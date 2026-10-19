package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

// Each kind of operation can be redone after an undo, and undone again after the redo.
func TestEveryOperationCanBeRedone(t *testing.T) {
	cases := []struct {
		name  string
		do    func(w *world) error
		after func(w *world) bool // true once the operation is in effect
	}{
		{"copy", func(w *world) error { _, err := w.ops.Copy([]string{w.path("src/a.txt")}, w.dst); return err },
			func(w *world) bool { return w.exists("dst/a.txt") }},
		{"move", func(w *world) error { _, err := w.ops.Move([]string{w.path("src/a.txt")}, w.dst); return err },
			func(w *world) bool { return w.exists("dst/a.txt") && !w.exists("src/a.txt") }},
		{"rename", func(w *world) error { _, err := w.ops.Rename(w.path("src/a.txt"), "b.txt"); return err },
			func(w *world) bool { return w.exists("src/b.txt") && !w.exists("src/a.txt") }},
		{"new folder", func(w *world) error { _, err := w.ops.Mkdir(w.dst, "new"); return err },
			func(w *world) bool { return w.exists("dst/new") }},
		{"new file", func(w *world) error { _, err := w.ops.Touch(w.dst, "new.txt"); return err },
			func(w *world) bool { return w.exists("dst/new.txt") }},
		{"trash", func(w *world) error { return w.ops.Trash([]string{w.path("src/a.txt")}) },
			func(w *world) bool { return !w.exists("src/a.txt") }},
		{"permissions", func(w *world) error { return w.ops.Chmod([]string{w.path("src/a.txt")}, 0o600) },
			func(w *world) bool { return mode(w.t, w.path("src/a.txt")) == 0o600 }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			w.must(os.Chmod(w.path("src/a.txt"), 0o644))
			w.must(c.do(w))
			if _, err := w.ops.Undo(); err != nil || c.after(w) {
				t.Fatalf("undo: %v, still in effect: %v", err, c.after(w))
			}
			if !w.ops.CanRedo() {
				t.Fatal("nothing to redo after an undo")
			}
			if _, err := w.ops.Redo(); err != nil || !c.after(w) {
				t.Fatalf("redo: %v, in effect: %v", err, c.after(w))
			}
			if _, err := w.ops.Undo(); err != nil || c.after(w) {
				t.Fatalf("undo after redo: %v, still in effect: %v", err, c.after(w))
			}
		})
	}
}

// A new operation after an undo drops what could be redone, as in every editor.
func TestANewOperationDropsTheRedos(t *testing.T) {
	w := newWorld(t)
	_, err := w.ops.Mkdir(w.dst, "one")
	w.must(err)
	_, err = w.ops.Undo()
	w.must(err)
	_, err = w.ops.Mkdir(w.dst, "two")
	w.must(err)
	if w.ops.CanRedo() {
		t.Fatal("a redo survived a new operation")
	}
	if _, err := w.ops.Redo(); err == nil {
		t.Fatal("Redo with nothing to redo must fail")
	}
}

// History lists what can be undone, newest first, and then what can be redone.
func TestHistoryListsTheUndosAndTheRedos(t *testing.T) {
	w := newWorld(t)
	_, err := w.ops.Mkdir(w.dst, "one")
	w.must(err)
	_, err = w.ops.Rename(w.path("src/a.txt"), "b.txt")
	w.must(err)
	_, err = w.ops.Touch(w.dst, "f.txt")
	w.must(err)
	_, err = w.ops.Undo()
	w.must(err)
	h := w.ops.History()
	want := []Entry{{Kind: KindRename, Subject: `"a.txt"`}, {Kind: KindMkdir, Subject: `"one"`}, {Kind: KindTouch, Subject: `"f.txt"`, Redo: true}}
	if len(h) != len(want) {
		t.Fatalf("history %+v", h)
	}
	for i := range want {
		if h[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, h[i], want[i])
		}
	}
	if filepath.Base(w.path("dst/one")) != "one" || !w.exists("dst/one") {
		t.Fatal("setup")
	}
}
