package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/termimg"
	"github.com/MathiasDrizzy/lazyfinder/internal/ui"
)

// sidebarFPS is how many times a second the pane of the herdr plugin may draw. Bubble Tea runs its renderer on a ticker
// of this rate for as long as the program lives, 60 by default, so an idle sidebar woke 60 times a second; at 15 a key
// or a click shows within 70 ms.
const sidebarFPS = 15

// programFPS is the rate the program draws at: low for the sidebar, 0 (Bubble Tea's default) for the window.
func programFPS(follow bool) int {
	if follow {
		return sidebarFPS
	}
	return 0
}

// signalGrace is how long a hangup or a kill waits for the program to end the way q does before it is killed.
const signalGrace = 10 * time.Second

// newProgram makes the program for m with filter (ui.QuitFilter) and stopOnSignal wired together, and returns it with
// the function that undoes stopOnSignal. A quit that filter sends to the model as a ui.SignalMsg came from a signal
// through Bubble Tea: it starts the grace time, and from then on a SIGINT counts too.
func newProgram(m tea.Model, filter func(tea.Model, tea.Msg) tea.Msg, grace time.Duration,
	opts ...tea.ProgramOption) (*tea.Program, func()) {
	quit := make(chan struct{})
	var once sync.Once
	quits := func(model tea.Model, msg tea.Msg) tea.Msg {
		msg = filter(model, msg)
		if _, ok := msg.(ui.SignalMsg); ok {
			once.Do(func() { close(quit) })
		}
		return msg
	}
	p := tea.NewProgram(m, append(opts, tea.WithFilter(quits))...)
	return p, stopOnSignal(p, grace, quit)
}

// stopOnSignal ends p on SIGHUP or SIGTERM, and on SIGINT once quit is closed, the way q does: the model is told first
// (ui.SignalMsg), so a copy under way is stopped and its half copy removed, and the terminal is given back in its
// normal mode with the deferred cleanups run. Closing quit says the model was already told. A program still running
// grace after the model was told is killed. Bubble Tea also answers SIGTERM and SIGINT, ignoring them while another
// program has the terminal, but it listens only once, so they are caught here too and a second one waits like the
// first. SIGINT waits for quit because ^C in a program that has the terminal (less) reaches lazyfinder too, and must
// not quit it. A signal that comes while the model has not read the first one (it is stuck, or an editor has the
// terminal), or after the program was killed, takes the default action: a hung lazyfinder still ends on a repeated
// kill. stop undoes it.
func stopOnSignal(p *tea.Program, grace time.Duration, quit <-chan struct{}) (stop func()) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	done := make(chan struct{})
	w := signalWatch{p: p, sig: sig, quit: quit, done: done}
	go func() {
		if w.told() && w.read() && w.wait(grace) {
			p.Kill()
			w.repeats()
		}
	}()
	return func() {
		signal.Stop(sig)
		close(done)
	}
}

// signalWatch follows the signals of one program; each step returns false once stop was called.
type signalWatch struct {
	p    *tea.Program
	sig  chan os.Signal
	quit <-chan struct{}
	done chan struct{}
	sent chan struct{} // closed once the model has read the SignalMsg sent here, nil if Bubble Tea told it
}

// counts says whether s asks to quit: SIGINT only once the model was told to.
func (w *signalWatch) counts(s os.Signal) bool {
	select {
	case <-w.quit:
		return true
	default:
		return s != syscall.SIGINT
	}
}

// told waits for the first signal that counts and tells the model, unless Bubble Tea did.
func (w *signalWatch) told() bool {
	for {
		select {
		case s := <-w.sig:
			if !w.counts(s) {
				continue
			}
			w.sent = make(chan struct{})
			go func() {
				w.p.Send(ui.SignalMsg{}) // returns once the program has read it, or has ended
				close(w.sent)
			}()
			return true
		case <-w.quit:
			select {
			case <-w.sig: // the same signal, which reached this channel too
			default:
			}
			return true
		case <-w.done:
			return false
		}
	}
}

// read waits for the model to read what was sent. While another program has the terminal (an editor run with
// tea.ExecProcess) it cannot, and a kill then would hand the terminal back to the shell under the editor, so the grace
// time starts only once it has. A signal repeated meanwhile ends a program that may be stuck.
func (w *signalWatch) read() bool {
	for w.sent != nil {
		select {
		case <-w.sent:
			return true
		case s := <-w.sig:
			if w.counts(s) {
				raise(s)
			}
		case <-w.done:
			return false
		}
	}
	return true
}

// wait gives the program grace to end; a signal meanwhile waits too, as the copy is being stopped.
func (w *signalWatch) wait(grace time.Duration) bool {
	timer := time.NewTimer(grace)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return true
		case <-w.sig:
		case <-w.done:
			return false
		}
	}
}

// repeats ends the process on any signal after the kill: a program that did not end is stuck.
func (w *signalWatch) repeats() {
	select {
	case s := <-w.sig:
		raise(s)
	case <-w.done:
	}
}

// raise gives s its default action, which ends the process, and sends it again.
func raise(s os.Signal) {
	signal.Reset(syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	_ = syscall.Kill(os.Getpid(), s.(syscall.Signal))
}

// clearImages deletes a preview image still in the terminal: a kill after signalGrace skips the model, which deletes it on
// every other way out.
func clearImages(out io.Writer, shown bool) {
	if shown {
		_, _ = fmt.Fprint(out, termimg.DeleteAllSequence())
	}
}
