// Package history keeps the back/forward stack of visited directories as an immutable value.
package history

// History is the visited paths and the position in them.
type History struct {
	paths []string
	pos   int
}

// New starts a history at start.
func New(start string) History { return History{paths: []string{start}} }

// Current is the path the user is in.
func (h History) Current() string { return h.paths[h.pos] }

// CanBack reports whether Back changes the path.
func (h History) CanBack() bool { return h.pos > 0 }

// CanForward reports whether Forward changes the path.
func (h History) CanForward() bool { return h.pos < len(h.paths)-1 }

// Visit records navigating to path and discards the forward stack. Visiting the current path is a no-op.
func (h History) Visit(path string) History {
	if path == h.Current() {
		return h
	}
	paths := make([]string, h.pos+1, h.pos+2)
	copy(paths, h.paths[:h.pos+1])
	return History{paths: append(paths, path), pos: h.pos + 1}
}

// Back moves one step back; at the start it returns h.
func (h History) Back() History {
	if !h.CanBack() {
		return h
	}
	return History{paths: h.paths, pos: h.pos - 1}
}

// Forward moves one step forward; at the end it returns h.
func (h History) Forward() History {
	if !h.CanForward() {
		return h
	}
	return History{paths: h.paths, pos: h.pos + 1}
}
