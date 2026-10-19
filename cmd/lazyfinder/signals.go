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

// hangupGrace is how long a hangup waits for the program to end the way q does before it is killed.
const hangupGrace = 10 * time.Second

// stopOnHangup ends p on SIGHUP the way q does: the model is told first (ui.SignalMsg), so a copy under way is stopped
// and its half copy removed, and the terminal is given back in its normal mode with the deferred cleanups run. A
// program still running grace after the model read it is killed. SIGTERM and SIGINT stay with Bubble Tea, which ignores
// them while another program has the terminal; ui.QuitFilter routes their quit to the model while a copy runs. stop
// undoes it.
func stopOnHangup(p *tea.Program, grace time.Duration) (stop func()) {
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	done := make(chan struct{})
	go func() {
		select {
		case <-hup:
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
		signal.Stop(hup)
		close(done)
	}
}

// clearImages deletes a preview image still in the terminal: a kill after hangupGrace skips the model, which deletes it on
// every other way out.
func clearImages(out io.Writer, shown bool) {
	if shown {
		_, _ = fmt.Fprint(out, termimg.DeleteAllSequence())
	}
}
