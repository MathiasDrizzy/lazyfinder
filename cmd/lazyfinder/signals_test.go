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

// started tells ready when the program has started m.
type started struct {
	tea.Model
	ready chan struct{}
}

func (s started) Init() tea.Cmd {
	close(s.ready)
	return s.Model.Init()
}

// holding keeps the program from reading messages for a while when it starts, the way a program run with
// tea.ExecProcess (an editor) does while it has the terminal; then it quits on a SignalMsg like polite.
type holding struct{ polite }

func (h holding) Init() tea.Cmd {
	time.Sleep(600 * time.Millisecond)
	return nil
}

// hangUp starts m with stopOnHangup, sends SIGHUP to the process and returns what Run returned, or fails the test
// if the program is still running after limit.
func hangUp(t *testing.T, m tea.Model, grace, limit time.Duration) error {
	t.Helper()
	ready := make(chan struct{})
	p := tea.NewProgram(started{m, ready}, tea.WithInput(strings.NewReader("")), tea.WithOutput(io.Discard))
	stop := stopOnHangup(p, grace)
	defer stop()
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	<-ready
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

// While another program has the terminal (an editor) the model cannot read the hangup; the grace time ran from the
// signal, so the kill gave the terminal back to the shell under the editor. It runs from when the model reads it.
func TestTheGraceTimeStartsWhenTheModelReadsTheHangup(t *testing.T) {
	told := make(chan struct{})
	if err := hangUp(t, holding{polite{told}}, 100*time.Millisecond, 3*time.Second); err != nil {
		t.Fatalf("err = %v, want the program to end the way q does", err)
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
