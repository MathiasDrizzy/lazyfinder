package ui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fileops"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// opTick is how often the status line shows the progress of a copy.
var opTick = 150 * time.Millisecond

type opKind int

const (
	opCopy opKind = iota
	opMove
	opDuplicate
)

// opState is a copy or a move running apart from the UI: its progress shows in the status line, Esc cancels it, and
// no other change to the files starts until it ends.
type opState struct {
	busy   bool
	kind   opKind
	paths  []string
	cancel context.CancelFunc
	done   *atomic.Int64 // bytes written so far
	total  *atomic.Int64 // bytes to write, once counted; 0 for a move that only renames
	quit   bool          // quit once it has stopped (the user quit while it ran)
}

type opDoneMsg struct {
	kind  opKind
	paths []string
	made  []string
	err   error
}

type opTickMsg struct{}

func opTickCmd() tea.Cmd { return tea.Tick(opTick, func(time.Time) tea.Msg { return opTickMsg{} }) }

// runOp starts kind on paths into dstDir apart from the UI.
func (m *Model) runOp(kind opKind, paths []string, dstDir string) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	done, total := new(atomic.Int64), new(atomic.Int64)
	m.op = opState{busy: true, kind: kind, paths: paths, cancel: cancel, done: done, total: total}
	m.setStatus(m.opProgress())
	ops := m.ops
	work := func() tea.Msg {
		defer cancel()
		onBytes := func(n int64) { done.Add(n) }
		msg := opDoneMsg{kind: kind, paths: paths}
		if kind == opMove {
			msg.made, msg.err = ops.MoveContext(ctx, paths, dstDir, onBytes)
			return msg
		}
		for _, p := range paths {
			if _, n, err := fileops.Count(ctx, p); err == nil {
				total.Add(n)
			}
		}
		msg.made, msg.err = ops.CopyContext(ctx, paths, dstDir, onBytes)
		return msg
	}
	return tea.Batch(work, opTickCmd())
}

// opProgress is the status line of the running operation.
func (m *Model) opProgress() string {
	format := i18n.T("Copiando %s…", "Copying %s…")
	switch m.op.kind {
	case opMove:
		format = i18n.T("Moviendo %s…", "Moving %s…")
	case opDuplicate:
		format = i18n.T("Duplicando %s…", "Duplicating %s…")
	}
	text := fmt.Sprintf(format, subject(m.op.paths))
	if total := m.op.total.Load(); total > 0 {
		text += fmt.Sprintf(" %s / %s", fsmodel.HumanSize(min(m.op.done.Load(), total)), fsmodel.HumanSize(total))
	}
	return text + "  " + i18n.T("(Esc cancela)", "(Esc cancels)")
}

func (m *Model) onOpTick() tea.Cmd {
	if !m.op.busy {
		return nil
	}
	if !m.statusErr {
		m.setStatus(m.opProgress())
	}
	return opTickCmd()
}

// cancelOp asks the running operation to stop; its end arrives as an opDoneMsg.
func (m *Model) cancelOp() {
	m.op.cancel()
	m.setStatus(i18n.T("Cancelando…", "Cancelling…"))
}

func (m *Model) onOpDone(msg opDoneMsg) tea.Cmd {
	quit := m.op.quit
	m.op = opState{}
	switch {
	case errors.Is(msg.err, context.Canceled):
		m.setStatus(i18n.T("Cancelado", "Cancelled"))
	case msg.err != nil:
		m.setError(msg.err)
	case msg.kind == opDuplicate:
		m.setStatus(fmt.Sprintf(i18n.T("Duplicado: %s", "Duplicated %s"), subject(msg.paths)))
	default:
		m.setStatus(fmt.Sprintf(i18n.T("Pegado: %s", "Pasted %s"), subject(msg.paths)))
	}
	if msg.kind == opMove && msg.err == nil && m.clip.cut && slices.Equal(m.clip.paths, msg.paths) {
		m.clip = clipboard{}
	}
	cmd := m.refresh(last(msg.made))
	if quit {
		return m.quit()
	}
	return cmd
}
