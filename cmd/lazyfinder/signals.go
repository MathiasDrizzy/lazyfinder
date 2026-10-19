package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/termimg"
	"github.com/MathiasDrizzy/lazyfinder/internal/ui"
)

// signalGrace is how long a hangup or a kill waits for the program to end the way q does before it is killed.
const signalGrace = 10 * time.Second

// stopOnSignal ends p on SIGHUP or SIGTERM the way q does: the model is told first (ui.SignalMsg), so a copy under way
// is stopped and its half copy removed, and the terminal is given back in its normal mode with the deferred cleanups
// run. A program still running grace after the model read it is killed. Bubble Tea also answers SIGTERM and SIGINT,
// ignoring them while another program has the terminal, and ui.QuitFilter routes its quit to the model while a copy
// runs; but it listens only once, so SIGTERM is caught here too and a second one does not take the default action,
// which kills the process in raw mode. stop undoes it.
func stopOnSignal(p *tea.Program, grace time.Duration) (stop func()) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		select {
		case <-sig:
		case <-done:
			return
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
