package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/termimg"
)

// quitOnHangup quits p on SIGHUP the way Bubble Tea does on SIGTERM, so the terminal is given back in its normal mode
// and the deferred cleanups run. stop undoes it.
func quitOnHangup(p *tea.Program) (stop func()) {
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	done := make(chan struct{})
	go func() {
		select {
		case <-hup:
			p.Quit()
		case <-done:
		}
	}()
	return func() {
		signal.Stop(hup)
		close(done)
	}
}

// clearImages deletes a preview image still in the terminal: a quit by signal skips the model, which deletes it on
// every other way out.
func clearImages(out io.Writer, shown bool) {
	if shown {
		_, _ = fmt.Fprint(out, termimg.DeleteAllSequence())
	}
}
