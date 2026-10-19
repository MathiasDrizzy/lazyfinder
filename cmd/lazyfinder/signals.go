package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/termimg"
	"github.com/MathiasDrizzy/lazyfinder/internal/ui"
)

// signalGrace is how long a hangup or a kill waits for the program to end the way q does before it is killed.
const signalGrace = 10 * time.Second

// newProgram makes the program for m with filter (ui.QuitFilter) and stopOnSignal wired together, and returns it with
// the function that undoes stopOnSignal. A SIGINT counts only once filter has sent a quit to the model as a
// ui.SignalMsg: from then on a SIGINT waits like a hangup instead of taking the default action, which kills the process
// in raw mode, once Bubble Tea has stopped listening.
func newProgram(m tea.Model, filter func(tea.Model, tea.Msg) tea.Msg, grace time.Duration,
	opts ...tea.ProgramOption) (*tea.Program, func()) {
	var armed atomic.Bool
	quits := func(model tea.Model, msg tea.Msg) tea.Msg {
		msg = filter(model, msg)
		if _, ok := msg.(ui.SignalMsg); ok {
			armed.Store(true)
		}
		return msg
	}
	p := tea.NewProgram(m, append(opts, tea.WithFilter(quits))...)
	return p, stopOnSignal(p, grace, &armed)
}

// stopOnSignal ends p on SIGHUP or SIGTERM, and on SIGINT once arm has been called, the way q does: the model is told
// first (ui.SignalMsg), so a copy under way is stopped and its half copy removed, and the terminal is given back in its
// normal mode with the deferred cleanups run. A program still running grace after the model read it is killed. Bubble
// Tea also answers SIGTERM and SIGINT, ignoring them while another program has the terminal, and ui.QuitFilter routes
// its quit to the model while a copy runs; but it listens only once, so they are caught here too and a second one does
// not take the default action. SIGINT waits for arm because ^C in a program that has the terminal (less) reaches
// lazyfinder too, and must not quit it. stop undoes it.
func stopOnSignal(p *tea.Program, grace time.Duration, armed *atomic.Bool) (stop func()) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	done := make(chan struct{})
	go func() {
		for waiting := true; waiting; {
			select {
			case s := <-sig:
				waiting = s == syscall.SIGINT && !armed.Load()
			case <-done:
				return
			}
		}
		// Send waits for the program to read it, or to end. While another program has the terminal (an editor run with
		// tea.ExecProcess) it is not read, and a kill then would hand the terminal back to the shell under the editor:
		// the grace time starts once the model has it.
		p.Send(ui.SignalMsg{})
		select {
		case <-time.After(grace):
			p.Kill()
		case <-done:
		}
	}()
	return func() {
		signal.Stop(sig)
		close(done)
	}
}

// clearImages deletes a preview image still in the terminal: a kill after signalGrace skips the model, which deletes it on
// every other way out.
func clearImages(out io.Writer, shown bool) {
	if shown {
		_, _ = fmt.Fprint(out, termimg.DeleteAllSequence())
	}
}
