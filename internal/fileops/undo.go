package fileops

import (
	"context"
	"errors"
	"fmt"
	"slices"
)

// Kind names an operation in the history; the UI words it in the user's language.
type Kind string

// The kinds of operation that can be undone.
const (
	KindCopy   Kind = "copy"
	KindMove   Kind = "move"
	KindRename Kind = "rename"
	KindMkdir  Kind = "new folder"
	KindTouch  Kind = "new file"
	KindTrash  Kind = "trashing"
	KindChmod  Kind = "permissions"
)

// step is one reversible operation. redo does it again through the same exported operation, which records a new
// step; it may be nil when the operation cannot be done again.
type step struct {
	kind    Kind
	subject string // what it acted on: a quoted name or "N items"
	undo    func(ctx context.Context, onBytes Progress) error
	redo    func(ctx context.Context, onBytes Progress) error
}

func (s step) desc() string { return fmt.Sprintf("%s of %s", s.kind, s.subject) }

// Entry is one line of the history: an operation that can be undone, or one an undo reversed that can be redone.
type Entry struct {
	Kind    Kind
	Subject string
	Redo    bool
}

// CanUndo says whether there is something to undo.
func (o *Ops) CanUndo() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.undo) > 0
}

// CanRedo says whether an undo can be redone.
func (o *Ops) CanRedo() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.redo) > 0
}

// push records s. A new operation drops what could be redone, unless it is a redo itself.
func (o *Ops) push(s step) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.undo = append(o.undo, s)
	o.pushed++
	if len(o.undo) > maxUndo {
		o.undo = o.undo[len(o.undo)-maxUndo:]
	}
	if !o.redoing {
		o.redo = nil
	}
}

// Undo reverses the last operation and returns what it undid. The operation can then be redone.
func (o *Ops) Undo() (string, error) { return o.UndoContext(context.Background(), nil) }

// UndoContext is Undo that tells onBytes what a move back across volumes writes and stops once ctx is cancelled. It
// runs the undo without the lock, like RedoContext; one that fails or stops stays to undo, from where it stopped.
func (o *Ops) UndoContext(ctx context.Context, onBytes Progress) (string, error) {
	return o.undoLast(ctx, onBytes, true)
}

// Retract undoes the last change and forgets it, with nothing to redo: for a change the user takes back before it is
// done, as Esc on the name of a new folder.
func (o *Ops) Retract() (string, error) { return o.undoLast(context.Background(), nil, false) }

func (o *Ops) undoLast(ctx context.Context, onBytes Progress, redoable bool) (string, error) {
	o.mu.Lock()
	if len(o.undo) == 0 {
		o.mu.Unlock()
		return "", errors.New("nothing to undo")
	}
	s := o.undo[len(o.undo)-1]
	o.undo = o.undo[:len(o.undo)-1]
	o.mu.Unlock()

	err := s.undo(ctx, onBytes)

	o.mu.Lock()
	defer o.mu.Unlock()
	if err != nil {
		o.undo = append(o.undo, s)
		return "", fmt.Errorf("undo %s: %w", s.desc(), err)
	}
	if redoable && s.redo != nil {
		o.redo = append(o.redo, s)
	}
	return s.desc(), nil
}

// Redo does again the operation the last undo reversed, and returns what it redid.
func (o *Ops) Redo() (string, error) { return o.RedoContext(context.Background(), nil) }

// RedoContext is Redo that tells onBytes what a copy writes and stops once ctx is cancelled, the way CopyContext
// does. It runs the operation without the lock, which records its own step. A redo that fails stays to redo, unless
// it did part of its work: that part is then an operation to undo.
func (o *Ops) RedoContext(ctx context.Context, onBytes Progress) (string, error) {
	o.mu.Lock()
	if len(o.redo) == 0 {
		o.mu.Unlock()
		return "", errors.New("nothing to redo")
	}
	s := o.redo[len(o.redo)-1]
	o.redo = o.redo[:len(o.redo)-1]
	o.redoing = true
	before := o.pushed
	o.mu.Unlock()

	err := s.redo(ctx, onBytes)

	o.mu.Lock()
	defer o.mu.Unlock()
	o.redoing = false
	if err != nil {
		if o.pushed == before {
			o.redo = append(o.redo, s)
		}
		return "", fmt.Errorf("redo %s: %w", s.desc(), err)
	}
	return s.desc(), nil
}

// History lists what can be undone, newest first, then what can be redone, next to redo first.
func (o *Ops) History() []Entry {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]Entry, 0, len(o.undo)+len(o.redo))
	for _, s := range slices.Backward(o.undo) {
		out = append(out, Entry{Kind: s.kind, Subject: s.subject})
	}
	for _, s := range slices.Backward(o.redo) {
		out = append(out, Entry{Kind: s.kind, Subject: s.subject, Redo: true})
	}
	return out
}
