package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fastOptions() Options {
	return Options{MaxDirs: 8, Poll: 50 * time.Millisecond, Debounce: 20 * time.Millisecond}
}

func expectEvent(t *testing.T, w Watcher, dir string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-w.Events():
			if ev.Dir == dir {
				return
			}
		case <-deadline:
			t.Fatalf("no event for %s within 2 s", dir)
		}
	}
}

func expectQuiet(t *testing.T, w Watcher, d time.Duration) {
	t.Helper()
	select {
	case ev := <-w.Events():
		t.Fatalf("unexpected event %v", ev)
	case <-time.After(d):
	}
}

func newFS(t *testing.T, o Options) *FS {
	t.Helper()
	w, err := NewFS(o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

func write(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCreatingAFileEmitsTheDirectory(t *testing.T) {
	dir := t.TempDir()
	w := newFS(t, fastOptions())
	if err := w.Watch(dir); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "new.txt"))
	expectEvent(t, w, dir)
}

func TestRemovingAndRenamingEmit(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a.txt"))
	w := newFS(t, fastOptions())
	if err := w.Watch(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")); err != nil {
		t.Fatal(err)
	}
	expectEvent(t, w, dir)
	if err := os.Remove(filepath.Join(dir, "b.txt")); err != nil {
		t.Fatal(err)
	}
	expectEvent(t, w, dir)
}

func TestABurstIsCoalesced(t *testing.T) {
	dir := t.TempDir()
	w := newFS(t, Options{MaxDirs: 8, Poll: time.Second, Debounce: 150 * time.Millisecond})
	if err := w.Watch(dir); err != nil {
		t.Fatal(err)
	}
	for i := range 50 {
		write(t, filepath.Join(dir, "f"+string(rune('a'+i%26))+string(rune('a'+i/26))))
	}
	expectEvent(t, w, dir)
	extra := 0
	for {
		select {
		case <-w.Events():
			extra++
			continue
		case <-time.After(400 * time.Millisecond):
		}
		break
	}
	if extra > 1 {
		t.Fatalf("a burst of 50 files produced %d extra events, want at most 1", extra)
	}
}

func TestNoiseIsIgnored(t *testing.T) {
	dir := t.TempDir()
	w := newFS(t, fastOptions())
	if err := w.Watch(dir); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{".DS_Store", "x.swp", ".x.swp", "draft~", ".#lock"} {
		write(t, filepath.Join(dir, n))
	}
	expectQuiet(t, w, 300*time.Millisecond)
	write(t, filepath.Join(dir, "real.txt"))
	expectEvent(t, w, dir)
}

func TestUnwatchStopsEvents(t *testing.T) {
	dir := t.TempDir()
	w := newFS(t, fastOptions())
	if err := w.Watch(dir); err != nil {
		t.Fatal(err)
	}
	w.Unwatch(dir)
	write(t, filepath.Join(dir, "late.txt"))
	expectQuiet(t, w, 300*time.Millisecond)
}

func TestWatchingTwiceIsHarmless(t *testing.T) {
	dir := t.TempDir()
	w := newFS(t, fastOptions())
	for range 3 {
		if err := w.Watch(dir); err != nil {
			t.Fatal(err)
		}
	}
	if got := w.Count(); got != 1 {
		t.Fatalf("watching the same dir three times counts %d, want 1", got)
	}
}

func TestBeyondMaxDirsFallsBackToPolling(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	w := newFS(t, Options{MaxDirs: 1, Poll: 40 * time.Millisecond, Debounce: 10 * time.Millisecond})
	if err := w.Watch(a); err != nil {
		t.Fatal(err)
	}
	if err := w.Watch(b); err != nil {
		t.Fatalf("the second dir must fall back to polling, got %v", err)
	}
	if w.Polling(a) || !w.Polling(b) {
		t.Fatalf("polling(a)=%v polling(b)=%v, want false and true", w.Polling(a), w.Polling(b))
	}
	write(t, filepath.Join(b, "polled.txt"))
	expectEvent(t, w, b)
}

func TestWatchingAMissingDirIsAnError(t *testing.T) {
	w := newFS(t, fastOptions())
	if err := w.Watch(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("a missing directory cannot be watched")
	}
}

func TestCloseClosesTheEventsChannel(t *testing.T) {
	w, err := NewFS(fastOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case _, ok := <-w.Events():
		if ok {
			t.Fatal("events must be closed after Close")
		}
	case <-time.After(time.Second):
		t.Fatal("Events was not closed")
	}
	if err := w.Close(); err != nil {
		t.Fatalf("a second Close must be harmless: %v", err)
	}
}

// Close racing the debounce timer used to close Events while flush was sending on it (found by -race in CI).
func TestCloseWhileAnEventIsBeingFlushedIsRaceFree(t *testing.T) {
	for i := 0; i < 200; i++ {
		dir := t.TempDir()
		w, err := NewFS(Options{MaxDirs: 8, Poll: time.Second, Debounce: time.Millisecond})
		if err != nil {
			t.Fatal(err)
		}
		if err := w.Watch(dir); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(dir, "a"))
		time.Sleep(time.Duration(i%4) * time.Millisecond) // lands the flush around Close
		_ = w.Close()
	}
}
