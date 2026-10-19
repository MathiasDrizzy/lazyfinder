package main

import (
	"bytes"
	"errors"
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

// hangUp starts m with stopOnHangup, sends SIGHUP to the process and returns what Run returned, or fails the test
// if the program is still running after limit.
func hangUp(t *testing.T, m tea.Model, grace, limit time.Duration) error {
	t.Helper()
	p := tea.NewProgram(m, tea.WithInput(strings.NewReader("")), tea.WithOutput(io.Discard))
	stop := stopOnHangup(p, grace)
	defer stop()
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	time.Sleep(100 * time.Millisecond) // let Run start
	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		return err
	case <-time.After(limit):
		t.Fatal("the program is still running after SIGHUP")
		return nil
	}
}

// A SIGHUP sent while the terminal is still there (kill -HUP, a multiplexer closing a pane) killed lazyfinder at once,
// leaving the terminal in raw mode and on the alternate screen; then it quit skipping the model, so a copy under way
// was cut off with its half copy left behind. It now reaches the model, which stops the copy and quits.
func TestAHangupReachesTheModelBeforeTheProgramEnds(t *testing.T) {
	told := make(chan struct{})
	if err := hangUp(t, polite{told}, time.Minute, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	select {
	case <-told:
	default:
		t.Fatal("the program ended without telling the model")
	}
}

// A model that does not end after a hangup (a copy that does not stop) is not waited for past the grace time.
func TestAProgramThatIgnoresTheHangupIsKilledAfterTheGraceTime(t *testing.T) {
	if err := hangUp(t, idle{}, 200*time.Millisecond, 3*time.Second); !errors.Is(err, tea.ErrProgramKilled) {
		t.Fatalf("err = %v, want the program killed", err)
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
