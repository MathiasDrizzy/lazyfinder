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

// stopGrace is how long a signal waits for the program to end the way q does before it is quit at once.
var stopGrace = 10 * time.Second

// stopOnSignal ends p on SIGHUP, SIGTERM or SIGINT the way q does: the model is told first, so a copy under way is
// stopped and its half copy removed, and the terminal is given back in its normal mode with the deferred cleanups run.
// A program still running stopGrace later is quit at once. Bubble Tea's own handler, which quits skipping the model,
// must be off (tea.WithoutSignalHandler). stop undoes it.
func stopOnSignal(p *tea.Program) (stop func()) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	done := make(chan struct{})
	go func() {
		select {
		case <-sig:
		case <-done:
			return
		}
		go p.Send(ui.SignalMsg{}) // Send waits for the program to read it, or to end
		select {
		case <-time.After(stopGrace):
			p.Quit()
		case <-done:
		}
	}()
	return func() {
		signal.Stop(sig)
		close(done)
	}
}

// clearImages deletes a preview image still in the terminal: a quit after stopGrace skips the model, which deletes it on
// every other way out.
func clearImages(out io.Writer, shown bool) {
	if shown {
		_, _ = fmt.Fprint(out, termimg.DeleteAllSequence())
	}
}
