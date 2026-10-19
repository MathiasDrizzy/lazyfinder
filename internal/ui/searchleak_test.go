package ui

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/search"
)

// A search that is replaced or closed must not leave the command that waits for its results blocked: Bubble Tea runs
// each command in a goroutine of its own, so a blocked one is a goroutine leaked per keystroke of the search bar.
func TestACancelledSearchReleasesItsWaitingCommand(t *testing.T) {
	e := newEnv(t)
	d := e.m.deps
	started := make(chan struct{})
	d.ContentFind = func(ctx context.Context, _, _ string, _ bool) ([]string, error) {
		close(started)
		<-ctx.Done() // a slow finder: ripgrep over a big tree
		return nil, ctx.Err()
	}
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	q, err := search.ParseBy("in:slow", e.m.deps.Now(), e.m.search.by)
	if err != nil {
		t.Fatal(err)
	}
	wait := e.m.startWalk(q)
	<-started
	e.m.search.cancel() // what a new keystroke or Esc does

	done := make(chan tea.Msg, 1)
	go func() { done <- wait() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the command that waits for the cancelled search is still blocked")
	}
}
