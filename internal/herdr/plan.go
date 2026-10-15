package herdr

import "encoding/json"

// The decisions of the plugin, without a socket: which tab an event is about, where the sidebar goes and which
// folder it follows.

// EventTab is the tab an event of herdr (HERDR_PLUGIN_EVENT_JSON) is about. The event names it, or a pane of it, or
// its workspace; else contextTab (HERDR_TAB_ID); else the tab of the focused pane. During a switch of workspace the
// focused pane is still the one being left, so the event wins over it.
func EventTab(event string, panes []Pane, contextTab string) string {
	var e struct {
		Data struct {
			TabID       string `json:"tab_id"`
			PaneID      string `json:"pane_id"`
			WorkspaceID string `json:"workspace_id"`
			Tab         struct {
				TabID string `json:"tab_id"`
			} `json:"tab"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(event), &e) // an empty or odd event leaves the fallbacks
	d := e.Data
	switch {
	case d.Tab.TabID != "":
		return d.Tab.TabID
	case d.TabID != "":
		return d.TabID
	case d.PaneID != "":
		for _, p := range panes {
			if p.ID == d.PaneID {
				return p.Tab
			}
		}
	case d.WorkspaceID != "":
		if t := tabOfWorkspace(panes, d.WorkspaceID); t != "" {
			return t
		}
	}
	if contextTab != "" {
		return contextTab
	}
	for _, p := range panes {
		if p.Focused {
			return p.Tab
		}
	}
	return ""
}

// tabOfWorkspace is the tab of the focused pane of a workspace, else of its first pane.
func tabOfWorkspace(panes []Pane, ws string) string {
	first := ""
	for _, p := range panes {
		if p.Workspace != ws {
			continue
		}
		if p.Focused {
			return p.Tab
		}
		if first == "" {
			first = p.Tab
		}
	}
	return first
}

// Sidebar is the lazyfinder pane of a tab.
func Sidebar(panes []Pane, tab string) (Pane, bool) {
	for _, p := range panes {
		if p.Tab == tab && p.Label == Label {
			return p, true
		}
	}
	return Pane{}, false
}

// Anchor is the pane the sidebar takes its folder from and gives the focus back to: the focused pane of the tab,
// else its first one, never the sidebar itself.
func Anchor(panes []Pane, tab string) (Pane, bool) {
	var first *Pane
	for i, p := range panes {
		if p.Tab != tab || p.Label == Label {
			continue
		}
		if p.Focused {
			return p, true
		}
		if first == nil {
			first = &panes[i]
		}
	}
	if first == nil {
		return Pane{}, false
	}
	return *first, true
}

// EdgePane is the pane of the layout on the dock side (left or right), the topmost one when the edge is split: the
// sidebar splits it and takes that edge.
func EdgePane(l Layout, dock string) string {
	var best *LayoutPane
	for i, p := range l.Panes {
		if best == nil {
			best = &l.Panes[i]
			continue
		}
		r, b := p.Rect, best.Rect
		var better bool
		if dock == "right" {
			better = r.X+r.Width > b.X+b.Width || (r.X+r.Width == b.X+b.Width && r.Y < b.Y)
		} else {
			better = r.X < b.X || (r.X == b.X && r.Y < b.Y)
		}
		if better {
			best = &l.Panes[i]
		}
	}
	if best == nil {
		return ""
	}
	return best.ID
}

// FollowDir is the folder the sidebar pane self follows: the one of the focused pane of its tab. Empty when the
// sidebar itself has the focus or self is unknown, so the folder shown stays.
func FollowDir(panes []Pane, self string) string {
	tab := ""
	for _, p := range panes {
		if p.ID == self {
			tab = p.Tab
		}
	}
	if tab == "" {
		return ""
	}
	for _, p := range panes {
		if p.Tab == tab && p.ID != self && p.Focused {
			return p.Dir()
		}
	}
	return ""
}

// resizeAmount is how far pane.resize moves the 50/50 split that plugin.pane.open makes, so the sidebar takes width
// percent of the tab.
func resizeAmount(width int) float64 {
	return float64(50-width) / 100
}
