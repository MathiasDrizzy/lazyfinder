// Package watch defines the live-follow seam. Milestone 1 ships only the interface and a no-op implementation; the
// fsnotify and git implementation arrives in milestone 2 (docs/adr/0003-live-watcher.md).
package watch

// Event reports that the content of Dir changed and its listing should be re-read.
type Event struct{ Dir string }

// Watcher watches directories and reports changes on Events.
type Watcher interface {
	Watch(dir string) error
	Unwatch(dir string)
	Events() <-chan Event
	Close() error
}

// Noop is the milestone 1 watcher: it watches nothing and never emits.
type Noop struct{}

// Watch accepts any directory.
func (Noop) Watch(string) error { return nil }

// Unwatch does nothing.
func (Noop) Unwatch(string) {}

// Events returns a channel that never delivers.
func (Noop) Events() <-chan Event { return nil }

// Close does nothing.
func (Noop) Close() error { return nil }
