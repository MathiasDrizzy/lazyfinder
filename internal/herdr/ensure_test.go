package herdr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fakeHerdr is a herdr with one tab: the panes it lists and every call it got, in order.
type fakeHerdr struct {
	mu    sync.Mutex
	panes []Pane
	calls []string
}

func (f *fakeHerdr) log() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Join(f.calls, "\n")
}

func (f *fakeHerdr) serve(t *testing.T) Client {
	return Client{Socket: fakeServer(t, func(method string, params json.RawMessage) string {
		f.mu.Lock()
		defer f.mu.Unlock()
		if method != "pane.list" && method != "pane.layout" {
			f.calls = append(f.calls, method+" "+string(params))
		}
		switch method {
		case "pane.list":
			b, _ := json.Marshal(map[string]any{"panes": f.panes})
			return `"result":` + string(b)
		case "pane.layout":
			return `"result":{"layout":{"panes":[{"pane_id":"w1:p1","rect":{"x":0,"y":0,"width":120,"height":40}}]}}`
		case "plugin.pane.open":
			f.panes = append(f.panes, Pane{ID: "w1:p9", Tab: "w1:t1", Label: Label})
			return `"result":{"plugin_pane":{"pane":{"pane_id":"w1:p9"}}}`
		case "pane.close":
			f.panes = f.panes[:1]
		}
		return `"result":{"type":"ok"}`
	})}
}

func newFake() *fakeHerdr {
	return &fakeHerdr{panes: []Pane{{ID: "w1:p1", Tab: "w1:t1", Workspace: "w1", Focused: true, Cwd: "/w/api"}}}
}

func envFor(t *testing.T) Env {
	return Env{TabID: "w1:t1", StateDir: t.TempDir(), Event: `{"event":"tab_created","data":{"tab":{"tab_id":"w1:t1"}}}`}
}

var left = Settings{Dock: "left", AutoOpen: true, Width: 25}

func TestEnsureOpensTheSidebarOnTheLeftAndGivesTheFocusBack(t *testing.T) {
	f := newFake()
	if err := Ensure(f.serve(t), envFor(t), left); err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		`plugin.pane.open {"cwd":"/w/api","direction":"right","entrypoint":"sidebar","focus":false,"placement":"split","plugin_id":"lazyfinder","target_pane_id":"w1:p1"}`,
		`pane.swap {"source_pane_id":"w1:p9","target_pane_id":"w1:p1"}`,
		`pane.resize {"amount":0.25,"direction":"left","pane_id":"w1:p9"}`,
		`pane.focus {"pane_id":"w1:p1"}`,
	}, "\n")
	if got := f.log(); got != want {
		t.Fatalf("calls:\n%s\nwant:\n%s", got, want)
	}
}

func TestEnsureOnTheRightNeedsNoSwap(t *testing.T) {
	f := newFake()
	s := left
	s.Dock, s.Width = "right", 40
	if err := Ensure(f.serve(t), envFor(t), s); err != nil {
		t.Fatal(err)
	}
	got := f.log()
	if strings.Contains(got, "pane.swap") || !strings.Contains(got, `pane.resize {"amount":0.1,"direction":"right","pane_id":"w1:p9"}`) {
		t.Fatalf("calls:\n%s", got)
	}
}

func TestEnsureIsIdempotent(t *testing.T) {
	f := newFake()
	c, e := f.serve(t), envFor(t)
	for range 3 {
		if err := Ensure(c, e, left); err != nil {
			t.Fatal(err)
		}
	}
	if n := strings.Count(f.log(), "plugin.pane.open"); n != 1 {
		t.Fatalf("opened %d sidebars:\n%s", n, f.log())
	}
}

func TestEnsureDoesNothingWithAutoOpenOff(t *testing.T) {
	f := newFake()
	s := left
	s.AutoOpen = false
	if err := Ensure(f.serve(t), envFor(t), s); err != nil || f.log() != "" {
		t.Fatalf("%v calls:\n%s", err, f.log())
	}
}

func TestToggleClosesAFocusedSidebarAndEnsureRespectsIt(t *testing.T) {
	f := newFake()
	c, e := f.serve(t), envFor(t)
	if err := Ensure(c, e, left); err != nil {
		t.Fatal(err)
	}
	f.panes[0].Focused, f.panes[1].Focused = false, true // the user is in the sidebar
	e.Event = ""
	if err := Toggle(c, e, left); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.log(), `pane.close {"pane_id":"w1:p9"}`) {
		t.Fatalf("not closed:\n%s", f.log())
	}
	f.panes[0].Focused = true
	before := strings.Count(f.log(), "plugin.pane.open")
	e1 := e
	e1.Event = `{"event":"tab_focused","data":{"tab_id":"w1:t1"}}`
	if err := Ensure(c, e1, left); err != nil {
		t.Fatal(err)
	}
	e2 := e
	e2.Event = `{"event":"pane_focused","data":{"pane_id":"w1:p1"}}`
	if err := Ensure(c, e2, left); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(f.log(), "plugin.pane.open"); n != before {
		t.Fatalf("a closed sidebar came back by itself:\n%s", f.log())
	}
	// the toggle opens it again, with the focus, and the hooks keep it
	if err := Toggle(c, e, left); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(f.log(), "plugin.pane.open"); n != before+1 || !strings.HasSuffix(f.log(), `pane.focus {"pane_id":"w1:p9"}`) {
		t.Fatalf("calls:\n%s", f.log())
	}
}

func TestToggleFocusesASidebarThatIsNotFocused(t *testing.T) {
	f := newFake()
	f.panes = append(f.panes, Pane{ID: "w1:p9", Tab: "w1:t1", Label: Label})
	if err := Toggle(f.serve(t), envFor(t), left); err != nil {
		t.Fatal(err)
	}
	if got := f.log(); got != `pane.focus {"pane_id":"w1:p9"}` {
		t.Fatalf("calls:\n%s", got)
	}
}

// A hook that finds the lock taken returns at once: the hooks that the plugin's own layout changes fire must not
// open a second sidebar.
func TestEnsureSkipsWhileAnotherHoldsTheLock(t *testing.T) {
	f := newFake()
	e := envFor(t)
	unlock, ok := lock(e.StateDir)
	if !ok {
		t.Fatal("no lock")
	}
	defer unlock()
	if err := Ensure(f.serve(t), e, left); err != nil || f.log() != "" {
		t.Fatalf("%v calls:\n%s", err, f.log())
	}
}

func TestSnoozeFileNamesAreSafe(t *testing.T) {
	dir := t.TempDir()
	if err := snooze(dir, "w1:t1/../x", true); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "snoozed"))
	if len(entries) != 1 || strings.ContainsAny(entries[0].Name(), "/:") {
		t.Fatalf("%v", entries)
	}
}
