package herdr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// fakeHerdr is a herdr with one tab: the panes it lists, the program each runs (lazyfinder for a pane of the plugin,
// else a shell, unless roots says otherwise) and every call it got, in order.
type fakeHerdr struct {
	mu    sync.Mutex
	panes []Pane
	roots map[string]string
	calls []string
	// swapped is set once the sidebar took the place of the pane it split: the layout lists it first
	swapped bool
}

// root is what pane id runs, as a spec: "lazyfinder" or "zsh" (that program alone), "child:micro" (lazyfinder with an
// editor in front) or "zsh+vim" (a shell with a program in front). A pane titled as the sidebar runs lazyfinder unless
// roots says otherwise.
func (f *fakeHerdr) root(id string) string {
	if r, ok := f.roots[id]; ok {
		return r
	}
	for _, p := range f.panes {
		if p.ID == id && p.Label == Label {
			return "lazyfinder"
		}
	}
	return "zsh"
}

// pid is the root process of pane id; the program in front of it, when it is not the root, has the next one.
func (f *fakeHerdr) pid(id string) int {
	for i, p := range f.panes {
		if p.ID == id {
			return 100 + 10*i
		}
	}
	return 1
}

// rootName is the program the process pid is: the root program of its pane.
func (f *fakeHerdr) rootName(pid int) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.panes {
		if f.pid(p.ID) != pid {
			continue
		}
		spec := f.root(p.ID)
		if _, ok := strings.CutPrefix(spec, "child:"); ok {
			return "lazyfinder"
		}
		name, _, _ := strings.Cut(spec, "+")
		return name
	}
	return ""
}

// processInfo is the pane.process_info result of pane id.
func (f *fakeHerdr) processInfo(id string) string {
	spec, root := f.root(id), f.pid(id)
	name, pid := spec, root
	if child, ok := strings.CutPrefix(spec, "child:"); ok {
		name, pid = child, root+1
	} else if _, front, ok := strings.Cut(spec, "+"); ok {
		name, pid = front, root+1
	}
	return fmt.Sprintf(`"result":{"process_info":{"shell_pid":%d,"foreground_processes":[{"name":%q,"pid":%d}]}}`, root, name, pid)
}

func (f *fakeHerdr) log() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Join(f.calls, "\n")
}

func (f *fakeHerdr) serve(t *testing.T) Client {
	return Client{RootName: f.rootName, Socket: fakeServer(t, func(method string, params json.RawMessage) string {
		f.mu.Lock()
		defer f.mu.Unlock()
		if method != "pane.list" && method != "pane.layout" && method != "pane.process_info" && method != "layout.export" {
			f.calls = append(f.calls, method+" "+string(params))
		}
		switch method {
		case "pane.list":
			b, _ := json.Marshal(map[string]any{"panes": f.panes})
			return `"result":` + string(b)
		case "pane.layout":
			var p struct {
				Pane string `json:"pane_id"`
			}
			_ = json.Unmarshal(params, &p)
			return fmt.Sprintf(`"result":{"layout":{"panes":[{"pane_id":%q,"rect":{"x":0,"y":0,"width":120,"height":40}}]}}`, p.Pane)
		case "plugin.pane.open":
			f.panes = append(f.panes, Pane{ID: "w1:p9", Tab: "w1:t1", Label: Label})
			return `"result":{"plugin_pane":{"pane":{"pane_id":"w1:p9"}}}`
		case "pane.process_info":
			var p struct {
				Pane string `json:"pane_id"`
			}
			_ = json.Unmarshal(params, &p)
			return f.processInfo(p.Pane)
		case "pane.swap":
			f.swapped = true
		case "layout.export":
			side, edge := `{"type":"pane","pane_id":"w1:p9"}`, `{"type":"pane","pane_id":"w1:p1"}`
			if !f.swapped {
				side, edge = edge, side
			}
			return `"result":{"layout":{"tab_id":"w1:t1","root":{"type":"split","direction":"right","ratio":0.5,"first":` + side + `,"second":` + edge + `}}}`
		case "pane.close":
			var p struct {
				Pane string `json:"pane_id"`
			}
			_ = json.Unmarshal(params, &p)
			f.panes = slices.DeleteFunc(f.panes, func(x Pane) bool { return x.ID == p.Pane })
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

var left = Settings{Dock: "left", AutoOpen: true, Columns: 30, ExternalColumns: 40, ExternalFrom: 200, Adapt: true, MinColumns: 22, FitNormal: true}

func TestEnsureOpensTheSidebarOnTheLeftAndGivesTheFocusBack(t *testing.T) {
	f := newFake()
	if err := Ensure(f.serve(t), envFor(t), left); err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		`plugin.pane.open {"cwd":"/w/api","direction":"right","entrypoint":"sidebar","focus":false,"placement":"split","plugin_id":"lazyfinder","target_pane_id":"w1:p1"}`,
		`pane.swap {"source_pane_id":"w1:p9","target_pane_id":"w1:p1"}`,
		`layout.set_split_ratio {"path":[],"ratio":0.2520833333333333,"tab_id":"w1:t1"}`,
		`pane.focus {"pane_id":"w1:p1"}`,
	}, "\n")
	if got := f.log(); got != want {
		t.Fatalf("calls:\n%s\nwant:\n%s", got, want)
	}
}

func TestEnsureOnTheRightNeedsNoSwap(t *testing.T) {
	f := newFake()
	s := left
	s.Dock, s.Columns = "right", 48
	if err := Ensure(f.serve(t), envFor(t), s); err != nil {
		t.Fatal(err)
	}
	got := f.log()
	if strings.Contains(got, "pane.swap") || !strings.Contains(got, `layout.set_split_ratio {"path":[],"ratio":0.6020833333333333,"tab_id":"w1:t1"}`) {
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
	if err := mark(dir, snoozed, "w1:t1/../x", true); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "snoozed"))
	if len(entries) != 1 || strings.ContainsAny(entries[0].Name(), "/:") {
		t.Fatalf("%v", entries)
	}
}

// Mathias, 2026-10-22: after "herdr server stop" and herdr again, some tabs had no sidebar and others had two. herdr
// restores the pane of the plugin as a shell that keeps the title "lazyfinder", which the hooks took for the sidebar.
// A pane with that title whose program is a shell is closed and a sidebar takes its place.
func TestEnsureReplacesASidebarThatCameBackAsAShell(t *testing.T) {
	f := newFake()
	f.panes = append(f.panes, Pane{ID: "w1:p7", Tab: "w1:t1", Label: Label})
	f.roots = map[string]string{"w1:p7": "zsh"}
	if err := Ensure(f.serve(t), envFor(t), left); err != nil {
		t.Fatal(err)
	}
	got := f.log()
	if !strings.HasPrefix(got, `pane.close {"pane_id":"w1:p7"}`) || strings.Count(got, "plugin.pane.open") != 1 {
		t.Fatalf("calls:\n%s", got)
	}
}

// A pane of the plugin running a program lazyfinder opened in it (an editor) is still the sidebar.
func TestASidebarRunningAnEditorIsKept(t *testing.T) {
	f := newFake()
	f.panes = append(f.panes, Pane{ID: "w1:p9", Tab: "w1:t1", Label: Label})
	f.roots = map[string]string{"w1:p9": "child:micro"}
	if err := Ensure(f.serve(t), envFor(t), left); err != nil || f.log() != "" {
		t.Fatalf("%v calls:\n%s", err, f.log())
	}
}

// The startup hook runs once herdr restored the session: every tab whose sidebar came back as a shell gets a sidebar
// again, without waiting for the user to visit it, and a tab whose sidebar the user closed stays without one.
func TestStartupRestoresTheSidebarOfEveryTab(t *testing.T) {
	f := newFake()
	f.panes = []Pane{
		{ID: "w1:p1", Tab: "w1:t1", Workspace: "w1", Focused: true, Cwd: "/w/api"},
		{ID: "w1:p2", Tab: "w1:t1", Workspace: "w1", Label: Label},
		{ID: "w2:p1", Tab: "w2:t1", Workspace: "w2", Cwd: "/w/web"},
		{ID: "w2:p2", Tab: "w2:t1", Workspace: "w2", Label: Label},
		{ID: "w3:p1", Tab: "w3:t1", Workspace: "w3", Cwd: "/w/docs"},
	}
	f.roots = map[string]string{"w1:p2": "zsh", "w2:p2": "zsh"}
	e := envFor(t)
	e.Event = ""
	if err := mark(e.StateDir, snoozed, "w3:t1", true); err != nil {
		t.Fatal(err)
	}
	if err := Startup(f.serve(t), e, left); err != nil {
		t.Fatal(err)
	}
	got := f.log()
	for _, want := range []string{`pane.close {"pane_id":"w1:p2"}`, `pane.close {"pane_id":"w2:p2"}`, `"target_pane_id":"w1:p1"`, `"target_pane_id":"w2:p1"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("no %s in:\n%s", want, got)
		}
	}
	if strings.Count(got, "plugin.pane.open") != 2 || strings.Contains(got, `"focus":true`) {
		t.Fatalf("calls:\n%s", got)
	}
}

// Mathias, 2026-10-22: "si salgo con q me cambio de space y despues vuelvo al mismo space se abre automaticamente,
// esto no deberia pasar porque me sali". Leaving the sidebar with q closes it in its tab as the toggle does; a new tab
// still gets one.
func TestHideKeepsTheSidebarClosedInItsTabOnly(t *testing.T) {
	f := newFake()
	f.panes = append(f.panes, Pane{ID: "w1:p9", Tab: "w1:t1", Label: Label})
	c, e := f.serve(t), envFor(t)
	if err := Hide(c, e, "w1:p9"); err != nil {
		t.Fatal(err)
	}
	f.panes = f.panes[:1] // lazyfinder quit and herdr closed its pane
	back := e
	back.Event = `{"event":"workspace_focused","data":{"workspace_id":"w1"}}`
	if err := Ensure(c, back, left); err != nil || strings.Contains(f.log(), "plugin.pane.open") {
		t.Fatalf("%v calls:\n%s", err, f.log())
	}
	f.panes = append(f.panes, Pane{ID: "w1:p5", Tab: "w1:t2", Workspace: "w1", Focused: true, Cwd: "/w/api"})
	f.panes[0].Focused = false
	next := e
	next.TabID, next.Event = "w1:t2", `{"event":"tab_created","data":{"tab":{"tab_id":"w1:t2"}}}`
	if err := Ensure(c, next, left); err != nil || !strings.Contains(f.log(), "plugin.pane.open") {
		t.Fatalf("%v calls:\n%s", err, f.log())
	}
}

// The mark of a closed tab goes once the tab is gone, so a later tab never inherits it.
func TestTheMarkOfATabThatIsGoneIsForgotten(t *testing.T) {
	f := newFake()
	e := envFor(t)
	if err := mark(e.StateDir, snoozed, "w9:t9", true); err != nil {
		t.Fatal(err)
	}
	if err := Ensure(f.serve(t), e, left); err != nil {
		t.Fatal(err)
	}
	if marked(e.StateDir, snoozed, "w9:t9") {
		t.Fatal("the mark of a tab that no longer exists stayed")
	}
}

// An agent pane is named after its folder by some plugins, so it can carry the title of the sidebar. It is not the
// sidebar: the toggle must never close it, and the tab still needs a sidebar of its own.
func agentTitledLikeTheSidebar() *fakeHerdr {
	f := newFake()
	f.panes[0].Label, f.panes[0].Agent = Label, "claude"
	return f
}

func TestToggleNeverClosesAnAgentPaneTitledLikeTheSidebar(t *testing.T) {
	f := agentTitledLikeTheSidebar()
	if err := Toggle(f.serve(t), envFor(t), left); err != nil {
		t.Fatal(err)
	}
	got := f.log()
	if strings.Contains(got, "pane.close") || strings.Contains(got, "tab.close") {
		t.Fatalf("closed the agent pane:\n%s", got)
	}
	if !strings.Contains(got, "plugin.pane.open") {
		t.Fatalf("no sidebar was opened:\n%s", got)
	}
}

func TestEnsureStillDocksASidebarNextToAnAgentPaneTitledLikeIt(t *testing.T) {
	f := agentTitledLikeTheSidebar()
	if err := Ensure(f.serve(t), envFor(t), left); err != nil {
		t.Fatal(err)
	}
	got := f.log()
	if strings.Contains(got, "pane.close") || !strings.Contains(got, "plugin.pane.open") {
		t.Fatalf("calls:\n%s", got)
	}
}

func TestTheAnchorOfASidebarMayBeAnAgentPaneTitledLikeIt(t *testing.T) {
	panes := []Pane{{ID: "w1:p1", Tab: "w1:t1", Focused: true, Label: Label, Agent: "claude"}}
	if _, ok := Sidebar(panes, "w1:t1"); ok {
		t.Fatal("an agent pane was taken for the sidebar")
	}
	if a, ok := Anchor(panes, "w1:t1"); !ok || a.ID != "w1:p1" {
		t.Fatalf("anchor = %v, %v", a, ok)
	}
}

// The sidebar is told by the program it runs, not by its title: a plugin that renames panes (herdr.auto-title once
// gave a sidebar the title "lazymark › lazyfinder") must not make the tab dock a second one.
func TestASidebarWithAnotherTitleIsStillTheSidebar(t *testing.T) {
	f := newFake()
	f.panes = append(f.panes, Pane{ID: "w1:p9", Tab: "w1:t1", Label: "lazymark › lazyfinder"})
	f.roots = map[string]string{"w1:p9": "lazyfinder"}
	if err := Ensure(f.serve(t), envFor(t), left); err != nil || f.log() != "" {
		t.Fatalf("%v calls:\n%s", err, f.log())
	}
}

// A pane of the user's that runs something else is never closed, whatever its title: the toggle opens the sidebar next
// to it instead. Only an idle shell with the title is taken for a sidebar herdr restored (the test above this one).
func TestToggleNeverClosesAFocusedPaneThatDoesNotRunLazyfinder(t *testing.T) {
	f := newFake()
	f.panes[0].Label = Label
	f.roots = map[string]string{"w1:p1": "zsh+vim"}
	if err := Toggle(f.serve(t), envFor(t), left); err != nil {
		t.Fatal(err)
	}
	got := f.log()
	if strings.Contains(got, "pane.close") || strings.Contains(got, "tab.close") || !strings.Contains(got, "plugin.pane.open") {
		t.Fatalf("calls:\n%s", got)
	}
}

// With the process as the proof, a pane that runs lazyfinder is closed by the toggle when it has the focus, even
// with another title.
func TestToggleClosesAFocusedSidebarWhateverItsTitle(t *testing.T) {
	f := newFake()
	f.panes[0].Focused = false
	f.panes = append(f.panes, Pane{ID: "w1:p9", Tab: "w1:t1", Label: "renamed", Focused: true})
	f.roots = map[string]string{"w1:p9": "lazyfinder"}
	if err := Toggle(f.serve(t), envFor(t), left); err != nil {
		t.Fatal(err)
	}
	if got := f.log(); !strings.HasPrefix(got, `pane.close {"pane_id":"w1:p9"}`) {
		t.Fatalf("calls:\n%s", got)
	}
}
