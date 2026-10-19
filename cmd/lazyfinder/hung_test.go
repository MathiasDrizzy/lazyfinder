package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/ui"
)

// hung stops reading messages for good: from the start ("init"), or on the quit it is told ("update").
type hung struct{ at string }

func (h hung) Init() tea.Cmd {
	fmt.Println("ready")
	if h.at == "init" {
		select {}
	}
	return nil
}

func (h hung) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(ui.SignalMsg); ok {
		select {}
	}
	return h, nil
}

func (hung) View() tea.View { return tea.NewView("") }

// TestHungHelper is not a test: it is the program run by TestAHungProgramEndsOnARepeatedSignal.
func TestHungHelper(t *testing.T) {
	at := os.Getenv("LAZYFINDER_HUNG_AT")
	if at == "" {
		t.Skip("run by TestAHungProgramEndsOnARepeatedSignal")
	}
	p, stop := newProgram(hung{at}, toModel, 200*time.Millisecond, tea.WithInput(strings.NewReader("")))
	defer stop()
	_, _ = p.Run()
}

// A program that no longer reads messages swallowed every SIGTERM once they were caught, so only SIGKILL ended it.
// A signal repeated while the model has not read the first, or after the grace time, takes the default action.
func TestAHungProgramEndsOnARepeatedSignal(t *testing.T) {
	for _, at := range []string{"init", "update"} {
		t.Run(at, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestHungHelper$")
			cmd.Env = append(os.Environ(), "LAZYFINDER_HUNG_AT="+at)
			out, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			if _, err := bufio.NewReader(out).ReadString('\n'); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			for range 2 {
				if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
					t.Fatal(err)
				}
				time.Sleep(400 * time.Millisecond)
			}
			select {
			case <-done:
				if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); !ok || !ws.Signaled() || ws.Signal() != syscall.SIGTERM {
					t.Fatalf("the program ended with %v, want killed by SIGTERM", cmd.ProcessState)
				}
			case <-time.After(3 * time.Second):
				_ = cmd.Process.Kill()
				<-done
				t.Fatal("a hung program outlived a repeated SIGTERM")
			}
		})
	}
}
