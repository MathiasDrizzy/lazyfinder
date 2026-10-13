package ui

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Sequences a program must send to give the terminal back: leave the alternate screen, show the cursor and stop
// mouse reporting.
var restoreSequences = []string{"\x1b[?1049l", "\x1b[?25h", "\x1b[?1002l"}

type panicModel struct{ *Model }

func (p panicModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "!" {
		panic("forced panic")
	}
	next, cmd := p.Model.Update(msg)
	p.Model = next.(*Model)
	return p, cmd
}

func runWithInput(t *testing.T, m tea.Model, input string) (string, error) {
	t.Helper()
	pr, pw := io.Pipe()
	var out bytes.Buffer
	p := tea.NewProgram(m, tea.WithInput(pr), tea.WithOutput(&out), tea.WithWindowSize(120, 40))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	time.Sleep(300 * time.Millisecond)
	_, _ = pw.Write([]byte(input))
	select {
	case err := <-done:
		_ = pw.Close()
		return out.String(), err
	case <-time.After(5 * time.Second):
		p.Kill()
		t.Fatal("the program did not exit")
	}
	return "", nil
}

func mustRestore(t *testing.T, out string) {
	t.Helper()
	for _, s := range restoreSequences {
		if !strings.Contains(out, s) {
			t.Errorf("the terminal was not restored: %q missing from the output", s)
		}
	}
}

func TestQuitRestoresTheTerminal(t *testing.T) {
	e := newEnv(t)
	out, err := runWithInput(t, e.m, "q")
	if err != nil {
		t.Fatalf("q must exit cleanly: %v", err)
	}
	mustRestore(t, out)
}

func TestAPanicRestoresTheTerminal(t *testing.T) {
	e := newEnv(t)
	out, err := runWithInput(t, panicModel{e.m}, "!")
	if err == nil {
		t.Fatal("a panic must come back as an error, not be swallowed")
	}
	mustRestore(t, out)
}
