package ui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fileops"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// progressWidth is the width in cells of the bar of a copy's progress.
const progressWidth = 16

// opTick is how often the status line shows the progress of a copy.
var opTick = 150 * time.Millisecond

type opKind int

const (
	opCopy opKind = iota
	opMove
	opDuplicate
	opRedo  // the last undo done again, which may be a copy or a move as long as the first
	opTrash // to the Trash, which copies when the Trash is on another volume
	opUndo  // the last operation reversed, which may move as much back
)

// opState is a copy, a move, a trash or an undo running apart from the UI: its progress shows in the status line, Esc cancels it, and
// no other change to the files starts until it ends.
type opState struct {
	busy   bool
	kind   opKind
	paths  []string
	cancel context.CancelFunc
	done   *atomic.Int64 // bytes written so far
	total  *atomic.Int64 // bytes to write, once counted; 0 for a move that only renames
	quit   bool          // quit once it has stopped (the user quit while it ran)
	ending bool          // asked to stop: the status line keeps saying so until it ends
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
		switch kind {
		case opRedo:
			_, msg.err = ops.RedoContext(ctx, onBytes)
			return msg
		case opUndo:
			_, msg.err = ops.UndoContext(ctx, onBytes)
			return msg
		case opTrash:
			msg.err = ops.TrashContext(ctx, paths, onBytes)
			return msg
		}
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
	var text string
	switch m.op.kind {
	case opRedo:
		text = i18n.T("Rehaciendo…", "Redoing…")
	case opUndo:
		text = i18n.T("Deshaciendo…", "Undoing…")
	case opTrash:
		text = fmt.Sprintf(i18n.T("Enviando %s a la Papelera…", "Moving %s to the Trash…"), subject(m.op.paths))
	default:
		text = fmt.Sprintf(format, subject(m.op.paths))
	}
	if total := m.op.total.Load(); total > 0 {
		done := min(m.op.done.Load(), total)
		full := int(done * progressWidth / total)
		bar := strings.Repeat("█", full) + strings.Repeat("░", progressWidth-full)
		text += fmt.Sprintf(" %s %s / %s", bar, fsmodel.HumanSize(done), fsmodel.HumanSize(total))
	}
	return text + "  " + i18n.T("(Esc cancela)", "(Esc cancels)")
}

func (m *Model) onOpTick() tea.Cmd {
	if !m.op.busy {
		return nil
	}
	if !m.statusErr && !m.op.ending {
		m.setStatus(m.opProgress())
	}
	return opTickCmd()
}

// cancelOp asks the running operation to stop; its end arrives as an opDoneMsg.
func (m *Model) cancelOp() {
	m.op.cancel()
	m.op.ending = true
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
	case msg.kind == opRedo:
		m.setStatus(i18n.T("Rehecho", "Redone"))
	case msg.kind == opUndo:
		m.setStatus(i18n.T("Deshecho", "Undone"))
	case msg.kind == opTrash:
		m.setStatus(fmt.Sprintf(i18n.T("%s enviado a la Papelera", "Moved %s to the Trash"), subject(msg.paths)))
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
