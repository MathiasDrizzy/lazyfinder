package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Options tunes the file system watcher.
type Options struct {
	MaxDirs  int           // directories watched natively; the rest are polled
	Poll     time.Duration // polling period for directories beyond MaxDirs
	Debounce time.Duration // trailing debounce that coalesces bursts
}

// DefaultOptions are the values of docs/adr/0003-live-watcher.md.
func DefaultOptions() Options {
	return Options{MaxDirs: 64, Poll: 2 * time.Second, Debounce: 50 * time.Millisecond}
}

// FS watches directories with fsnotify and falls back to polling beyond MaxDirs or when fsnotify refuses a directory.
type FS struct {
	opt Options
	fsn *fsnotify.Watcher
	out chan Event

	mu      sync.Mutex
	native  map[string]bool     // directories watched by fsnotify
	polled  map[string]string   // directory -> listing signature
	pending map[string]struct{} // directories waiting for the debounce
	timer   *time.Timer
	done    chan struct{}
	closed  bool
	wg      sync.WaitGroup
}

// NewFS starts a watcher.
func NewFS(opt Options) (*FS, error) {
	def := DefaultOptions()
	if opt.MaxDirs <= 0 {
		opt.MaxDirs = def.MaxDirs
	}
	if opt.Poll <= 0 {
		opt.Poll = def.Poll
	}
	if opt.Debounce <= 0 {
		opt.Debounce = def.Debounce
	}
	fsn, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("start the file watcher: %w", err)
	}
	w := &FS{
		opt: opt, fsn: fsn, out: make(chan Event, 16),
		native: map[string]bool{}, polled: map[string]string{}, pending: map[string]struct{}{},
		done: make(chan struct{}),
	}
	w.wg.Add(2)
	go w.readLoop()
	go w.pollLoop()
	return w, nil
}

// Events delivers one Event per changed directory after the debounce; it is closed by Close.
func (w *FS) Events() <-chan Event { return w.out }

// Count is the number of directories being watched, natively or by polling.
func (w *FS) Count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.native) + len(w.polled)
}

// Polling reports whether dir is watched by polling instead of fsnotify.
func (w *FS) Polling(dir string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, ok := w.polled[filepath.Clean(dir)]
	return ok
}

// Watch starts following dir. Watching a directory again is harmless.
func (w *FS) Watch(dir string) error {
	dir = filepath.Clean(dir)
	sig, err := signature(dir)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	if w.native[dir] {
		return nil
	}
	if _, ok := w.polled[dir]; ok {
		return nil
	}
	if len(w.native) < w.opt.MaxDirs && w.fsn.Add(dir) == nil {
		w.native[dir] = true
		return nil
	}
	w.polled[dir] = sig
	return nil
}

// Unwatch stops following dir.
func (w *FS) Unwatch(dir string) {
	dir = filepath.Clean(dir)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.native[dir] {
		_ = w.fsn.Remove(dir)
		delete(w.native, dir)
	}
	delete(w.polled, dir)
	delete(w.pending, dir)
}

// Close stops the watcher and closes Events. It can be called more than once.
func (w *FS) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	close(w.done)
	if w.timer != nil {
		w.timer.Stop()
	}
	w.mu.Unlock()
	err := w.fsn.Close()
	w.wg.Wait()
	close(w.out)
	return err
}

func (w *FS) readLoop() {
	defer w.wg.Done()
	for {
		select {
		case ev, ok := <-w.fsn.Events:
			if !ok {
				return
			}
			if noise(filepath.Base(ev.Name)) {
				continue
			}
			w.mark(filepath.Dir(ev.Name))
		case _, ok := <-w.fsn.Errors:
			if !ok {
				return
			}
		case <-w.done:
			return
		}
	}
}

func (w *FS) pollLoop() {
	defer w.wg.Done()
	tick := time.NewTicker(w.opt.Poll)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			w.pollOnce()
		case <-w.done:
			return
		}
	}
}

func (w *FS) pollOnce() {
	w.mu.Lock()
	dirs := make([]string, 0, len(w.polled))
	for d := range w.polled {
		dirs = append(dirs, d)
	}
	w.mu.Unlock()
	for _, d := range dirs {
		sig, err := signature(d)
		if err != nil {
			sig = "gone"
		}
		w.mu.Lock()
		old, still := w.polled[d]
		changed := still && old != sig
		if changed {
			w.polled[d] = sig
		}
		w.mu.Unlock()
		if changed {
			w.mark(d)
		}
	}
}

// mark queues dir for the debounced event.
func (w *FS) mark(dir string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || !(w.native[dir] || w.polledLocked(dir)) {
		return
	}
	w.pending[dir] = struct{}{}
	if w.timer == nil {
		w.timer = time.AfterFunc(w.opt.Debounce, w.flush)
		return
	}
	w.timer.Reset(w.opt.Debounce)
}

func (w *FS) polledLocked(dir string) bool { _, ok := w.polled[dir]; return ok }

func (w *FS) flush() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	dirs := make([]string, 0, len(w.pending))
	for d := range w.pending {
		dirs = append(dirs, d)
	}
	w.pending = map[string]struct{}{}
	w.mu.Unlock()
	for _, d := range dirs {
		select {
		case w.out <- Event{Dir: d}:
		case <-w.done:
			return
		}
	}
}

// noise reports names that editors and the system churn and that never matter to a listing.
func noise(name string) bool {
	switch {
	case name == ".DS_Store":
		return true
	case strings.HasSuffix(name, ".swp"), strings.HasSuffix(name, ".swx"), strings.HasSuffix(name, "~"):
		return true
	case strings.HasPrefix(name, ".#"):
		return true
	}
	return false
}

// signature fingerprints a directory listing (names, sizes, modification times) for the polling fallback.
func signature(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range entries {
		if noise(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "%s|%d|%d\n", e.Name(), info.Size(), info.ModTime().UnixNano())
	}
	return b.String(), nil
}
