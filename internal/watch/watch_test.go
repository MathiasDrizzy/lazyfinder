package watch

import "testing"

func TestNoopWatcherNeverEmitsAndAcceptsAnyDir(t *testing.T) {
	var w Watcher = Noop{}
	if err := w.Watch("/anything"); err != nil {
		t.Fatal(err)
	}
	w.Unwatch("/anything")
	select {
	case ev := <-w.Events():
		t.Fatalf("unexpected event %v", ev)
	default:
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}
