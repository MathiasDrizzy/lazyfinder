package herdr

import "testing"

// Order 2026-10-15-16: «que siga la carpeta donde esta el space y, cuando se inicie tiene que estar a la izquierda o
// derecha (se modifica en configuracion, se abre automaticamente…)». These are the decisions, without a socket.

var tabPanes = []Pane{
	{ID: "w1:p1", Tab: "w1:t1", Workspace: "w1", Focused: true, Cwd: "/w/api", ForegroundCwd: "/w/api/cmd"},
	{ID: "w1:p2", Tab: "w1:t1", Workspace: "w1", Cwd: "/w/api"},
	{ID: "w2:p1", Tab: "w2:t1", Workspace: "w2", Cwd: "/w/web"},
}

func TestEventTabComesFromTheEventFirst(t *testing.T) {
	cases := []struct{ name, event, ctx, want string }{
		{"tab created", `{"event":"tab_created","data":{"tab":{"tab_id":"w2:t1","workspace_id":"w2"}}}`, "w1:t1", "w2:t1"},
		{"pane focused", `{"event":"pane_focused","data":{"pane_id":"w2:p1","workspace_id":"w2"}}`, "w1:t1", "w2:t1"},
		{"workspace focused", `{"event":"workspace_focused","data":{"workspace_id":"w2"}}`, "", "w2:t1"},
		{"only the context", `{"event":"tab_focused","data":{}}`, "w1:t1", "w1:t1"},
		{"no event: the focused tab", ``, "", "w1:t1"},
	}
	for _, c := range cases {
		if got := EventTab(c.event, tabPanes, c.ctx); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestTheSidebarIsThePaneWithTheLabel(t *testing.T) {
	panes := append([]Pane{{ID: "w1:p3", Tab: "w1:t1", Label: Label}}, tabPanes...)
	if s, ok := Sidebar(panes, "w1:t1"); !ok || s.ID != "w1:p3" {
		t.Fatalf("sidebar %+v %v", s, ok)
	}
	if _, ok := Sidebar(panes, "w2:t1"); ok {
		t.Fatal("the other tab has no sidebar")
	}
}

func TestTheAnchorIsTheFocusedPaneOfTheTabAndNeverTheSidebar(t *testing.T) {
	if a, ok := Anchor(tabPanes, "w1:t1"); !ok || a.ID != "w1:p1" {
		t.Fatalf("anchor %+v", a)
	}
	if a, ok := Anchor(tabPanes, "w2:t1"); !ok || a.ID != "w2:p1" {
		t.Fatalf("no focused pane in the tab: the first one, got %+v", a)
	}
	side := []Pane{{ID: "w1:p3", Tab: "w1:t1", Label: Label, Focused: true}, {ID: "w1:p1", Tab: "w1:t1"}}
	if a, ok := Anchor(side, "w1:t1"); !ok || a.ID != "w1:p1" {
		t.Fatalf("anchor %+v", a)
	}
}

func TestDirPrefersTheForegroundFolder(t *testing.T) {
	if d := tabPanes[0].Dir(); d != "/w/api/cmd" {
		t.Fatal(d)
	}
	if d := tabPanes[1].Dir(); d != "/w/api" {
		t.Fatal(d)
	}
}

func TestTheEdgePaneIsTheTopmostOneOnTheDockSide(t *testing.T) {
	l := Layout{Panes: []LayoutPane{
		{ID: "a", Rect: Rect{X: 0, Y: 0, Width: 60, Height: 20}},
		{ID: "b", Rect: Rect{X: 0, Y: 20, Width: 60, Height: 20}},
		{ID: "c", Rect: Rect{X: 60, Y: 0, Width: 60, Height: 40}},
	}}
	if got := EdgePane(l, "left"); got != "a" {
		t.Errorf("left: %q", got)
	}
	if got := EdgePane(l, "right"); got != "c" {
		t.Errorf("right: %q", got)
	}
}

func TestTheSidebarFollowsTheFocusedPaneOfItsTab(t *testing.T) {
	panes := append([]Pane{{ID: "w1:p3", Tab: "w1:t1", Label: Label}}, tabPanes...)
	if d := FollowDir(panes, "w1:p3"); d != "/w/api/cmd" {
		t.Fatalf("follow %q", d)
	}
	// with the sidebar focused nothing else is: keep the folder shown
	panes[0].Focused, panes[1].Focused = true, false
	if d := FollowDir(panes, "w1:p3"); d != "" {
		t.Fatalf("follow %q, want nothing", d)
	}
	if d := FollowDir(panes, "w9:p9"); d != "" {
		t.Fatalf("unknown pane: %q", d)
	}
}

func TestResizeAmountMovesTheSplitToTheWidth(t *testing.T) {
	if a := resizeAmount(25); a != 0.25 {
		t.Fatal(a)
	}
	if a := resizeAmount(50); a != 0 {
		t.Fatal(a)
	}
}
