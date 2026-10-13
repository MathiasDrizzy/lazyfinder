package history

import "testing"

func TestBackForward(t *testing.T) {
	h := New("/a")
	h = h.Visit("/a/b").Visit("/a/b/c")
	if h.Current() != "/a/b/c" || !h.CanBack() || h.CanForward() {
		t.Fatalf("unexpected state: %+v", h)
	}
	h = h.Back()
	if h.Current() != "/a/b" || !h.CanForward() {
		t.Fatalf("after back: %+v", h)
	}
	h = h.Back().Back() // clamps at the start
	if h.Current() != "/a" || h.CanBack() {
		t.Fatalf("after clamp: %+v", h)
	}
	h = h.Forward().Forward()
	if h.Current() != "/a/b/c" || h.CanForward() {
		t.Fatalf("after forward: %+v", h)
	}
}

func TestVisitAfterBackDiscardsForward(t *testing.T) {
	h := New("/a").Visit("/b").Visit("/c").Back().Visit("/d")
	if h.CanForward() || h.Current() != "/d" {
		t.Fatalf("forward stack must be discarded: %+v", h)
	}
	if h.Back().Current() != "/b" {
		t.Fatalf("back from /d should be /b, got %q", h.Back().Current())
	}
}

func TestVisitSamePathIsNoop(t *testing.T) {
	h := New("/a").Visit("/a")
	if h.CanBack() {
		t.Fatal("revisiting the current path must not add history")
	}
}

func TestImmutable(t *testing.T) {
	h := New("/a")
	_ = h.Visit("/b")
	if h.Current() != "/a" || h.CanForward() {
		t.Fatal("Visit mutated its receiver")
	}
}
