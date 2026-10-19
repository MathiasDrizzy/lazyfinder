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
	"github.com/MathiasDrizzy/lazyfinder/internal/ui"
)

type idle struct{}

func (idle) Init() tea.Cmd                       { return nil }
func (idle) Update(tea.Msg) (tea.Model, tea.Cmd) { return idle{}, nil }
func (idle) View() tea.View                      { return tea.NewView("") }

// polite quits when it is told about the signal, as the real model does once a copy under way has stopped.
type polite struct{ told chan<- struct{} }

func (polite) Init() tea.Cmd { return nil }
func (p polite) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(ui.SignalMsg); ok {
		close(p.told)
		return p, tea.Quit
	}
	return p, nil
}
func (polite) View() tea.View { return tea.NewView("") }

// runSignalled starts m with stopOnSignal, sends sig to the process and says whether the program ended within limit.
func runSignalled(t *testing.T, m tea.Model, sig syscall.Signal, limit time.Duration) bool {
	t.Helper()
	p := tea.NewProgram(m, tea.WithInput(strings.NewReader("")), tea.WithOutput(io.Discard), tea.WithoutSignalHandler())
	stop := stopOnSignal(p)
	defer stop()
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	time.Sleep(100 * time.Millisecond) // let Run start
	if err := syscall.Kill(os.Getpid(), sig); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
		return true
	case <-time.After(limit):
		return false
	}
}

// A SIGHUP sent while the terminal is still there (kill -HUP, a multiplexer closing a pane) killed lazyfinder at once,
// leaving the terminal in raw mode and on the alternate screen; then it quit skipping the model, so a copy under way
// was cut off with its half copy left behind. SIGHUP and SIGTERM now reach the model, which stops the copy and quits.
func TestASignalReachesTheModelBeforeTheProgramEnds(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGHUP, syscall.SIGTERM} {
		told := make(chan struct{})
		if !runSignalled(t, polite{told}, sig, 3*time.Second) {
			t.Fatalf("%v: the program is still running", sig)
		}
		select {
		case <-told:
		default:
			t.Fatalf("%v: the program ended without telling the model", sig)
		}
	}
}

// A model that does not end after a signal (a copy that does not stop) is not waited for past the grace time.
func TestAProgramThatIgnoresTheSignalEndsAfterTheGraceTime(t *testing.T) {
	old := stopGrace
	stopGrace = 200 * time.Millisecond
	t.Cleanup(func() { stopGrace = old })
	if !runSignalled(t, idle{}, syscall.SIGHUP, 3*time.Second) {
		t.Fatal("the program is still running after the grace time")
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
