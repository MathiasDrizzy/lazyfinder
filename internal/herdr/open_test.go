package herdr

import (
	"encoding/json"
	"strings"
	"testing"
)

// tabsHerdr is a herdr that creates tabs: each new tab gets one shell pane, and closing a tab drops its panes.
type tabsHerdr struct{ fakeHerdr }

func (f *tabsHerdr) serve(t *testing.T) Client {
	return Client{Socket: fakeServer(t, func(method string, params json.RawMessage) string {
		f.mu.Lock()
		defer f.mu.Unlock()
		if method != "pane.list" && method != "pane.process_info" {
			f.calls = append(f.calls, method+" "+string(params))
		}
		switch method {
		case "pane.list":
			b, _ := json.Marshal(map[string]any{"panes": f.panes})
			return `"result":` + string(b)
		case "pane.process_info":
			var p struct {
				Pane string `json:"pane_id"`
			}
			_ = json.Unmarshal(params, &p)
			return `"result":{"process_info":{"shell_pid":7,"foreground_processes":[{"name":"` + f.root(p.Pane) + `","pid":7}]}}`
		case "tab.create":
			f.panes = append(f.panes, Pane{ID: "w1:p5", Tab: "w1:t2", Workspace: "w1"})
			return `"result":{"type":"tab_created","tab":{"tab_id":"w1:t2"},"root_pane":{"pane_id":"w1:p5","tab_id":"w1:t2"}}`
		case "tab.close":
			var p struct {
				Tab string `json:"tab_id"`
			}
			_ = json.Unmarshal(params, &p)
			var kept []Pane
			for _, x := range f.panes {
				if x.Tab != p.Tab {
					kept = append(kept, x)
				}
			}
			f.panes = kept
		}
		return `"result":{"type":"ok"}`
	})}
}

// Mathias, 2026-10-22: "click y saltar a otra pestaña de herdr con el programa configurado". A terminal program opens
// in a new tab of herdr, in the folder of the file, and takes the place of its shell, so quitting it ends the pane.
func TestOpenRunsTheProgramInANewTab(t *testing.T) {
	f := &tabsHerdr{*newFake()}
	e := envFor(t)
	if err := Open(f.serve(t), e, "/w/my notes", []string{"micro", "/w/my notes/it's.md"}, "micro"); err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		`tab.create {"cwd":"/w/my notes","focus":true,"label":"micro"}`,
		`pane.send_text {"pane_id":"w1:p5","text":"exec 'micro' '/w/my notes/it'\\''s.md'\n"}`,
	}, "\n")
	if got := f.log(); got != want {
		t.Fatalf("calls:\n%s\nwant:\n%s", got, want)
	}
	if !marked(e.StateDir, opened, "w1:t2") {
		t.Fatal("the tab is not marked as opened by lazyfinder")
	}
}

// When the program quits, its tab is left with the sidebar the hooks docked there: that tab closes. A tab of the user
// keeps its sidebar even when it is the only pane.
func TestATabOpenedForAProgramClosesWithIt(t *testing.T) {
	f := &tabsHerdr{*newFake()}
	f.panes = []Pane{
		{ID: "w1:p1", Tab: "w1:t1", Label: Label},
		{ID: "w1:p6", Tab: "w1:t2", Label: Label},
		{ID: "w1:p7", Tab: "w1:t3", Label: Label},
		{ID: "w1:p8", Tab: "w1:t3"},
	}
	e := envFor(t)
	for _, tab := range []string{"w1:t2", "w1:t3"} {
		if err := mark(e.StateDir, opened, tab, true); err != nil {
			t.Fatal(err)
		}
	}
	if err := Ensure(f.serve(t), e, left); err != nil {
		t.Fatal(err)
	}
	if got := f.log(); got != `tab.close {"tab_id":"w1:t2"}` {
		t.Fatalf("calls:\n%s", got)
	}
}
