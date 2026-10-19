package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/termimg"
)

type idle struct{}

func (idle) Init() tea.Cmd                       { return nil }
func (idle) Update(tea.Msg) (tea.Model, tea.Cmd) { return idle{}, nil }
func (idle) View() tea.View                      { return tea.NewView("") }

// A SIGHUP sent while the terminal is still there (kill -HUP, a multiplexer closing a pane) killed lazyfinder at once,
// leaving the terminal in raw mode and on the alternate screen. It now quits the way SIGTERM does.
func TestAHangupQuitsTheProgramInsteadOfKillingIt(t *testing.T) {
	p := tea.NewProgram(idle{}, tea.WithInput(strings.NewReader("")), tea.WithOutput(io.Discard))
	stop := quitOnHangup(p)
	defer stop()
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	time.Sleep(100 * time.Millisecond) // let Run start
	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the program is still running after SIGHUP")
	}
}

func TestAnImageLeftOnScreenIsDeletedAfterTheProgram(t *testing.T) {
	var out bytes.Buffer
	clearImages(&out, true)
	if out.String() != termimg.DeleteAllSequence() {
		t.Fatalf("wrote %q", out.String())
	}
	out.Reset()
	clearImages(&out, false)
	if out.Len() != 0 {
		t.Fatalf("wrote %q without an image", out.String())
	}
}
