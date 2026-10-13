package ui

import "testing"

func TestUpOnTheFirstRowGoesToTheLastAndDownOnTheLastToTheFirst(t *testing.T) {
	e := newEnv(t)
	last := len(e.m.entries()) - 1
	e.key("up")
	if e.m.sel != last {
		t.Fatalf("up on the first row: sel = %d, want %d", e.m.sel, last)
	}
	e.key("down")
	if e.m.sel != 0 {
		t.Fatalf("down on the last row: sel = %d, want 0", e.m.sel)
	}
	e.key(",")
	e.key("up")
	if e.m.set.sel != settingCount-1 {
		t.Fatalf("settings: up on the first row gave %d, want %d", e.m.set.sel, settingCount-1)
	}
	e.key("down")
	if e.m.set.sel != 0 {
		t.Fatalf("settings: down on the last row gave %d", e.m.set.sel)
	}
	e.key("end")
	if e.m.set.sel != settingCount-1 {
		t.Fatalf("settings: end gave %d", e.m.set.sel)
	}
}

func TestWrapKeepsACursorInsideTheList(t *testing.T) {
	for _, c := range []struct{ i, n, want int }{{-1, 5, 4}, {5, 5, 0}, {2, 5, 2}, {0, 0, 0}, {-1, 0, 0}} {
		if got := wrap(c.i, c.n); got != c.want {
			t.Errorf("wrap(%d,%d) = %d, want %d", c.i, c.n, got, c.want)
		}
	}
}
