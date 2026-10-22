package herdr

import (
	"errors"
	"fmt"
)

// NormalColumns is the width the normal mode needs: 60 columns of lazyfinder and the border herdr puts around a pane.
const NormalColumns = 62

// Node is a node of the tree of splits layout.export gives: a pane, or a split of two nodes.
type Node struct {
	Type      string  `json:"type"`      // pane or split
	Direction string  `json:"direction"` // right (side by side) or down
	Ratio     float64 `json:"ratio"`     // share of the first node
	First     *Node   `json:"first,omitempty"`
	Second    *Node   `json:"second,omitempty"`
	Pane      string  `json:"pane_id,omitempty"`
}

// WidthFor is the width of the sidebar when herdr is screen columns wide.
func (s Settings) WidthFor(screen int) int {
	w := s.Columns
	if screen >= s.ExternalFrom {
		w = s.ExternalColumns
	}
	return max(w, s.MinColumns)
}

// tabOf is the tab of a pane.
func tabOf(c Client, pane string) (string, error) {
	panes, err := c.Panes()
	if err != nil {
		return "", err
	}
	for _, p := range panes {
		if p.ID == pane {
			return p.Tab, nil
		}
	}
	return "", fmt.Errorf("herdr: no pane %s", pane)
}

// widths are the width of the tab of pane and of pane itself, in columns.
func widths(c Client, pane string) (tab, own int, err error) {
	l, err := c.LayoutOf(pane)
	if err != nil {
		return 0, 0, err
	}
	for _, p := range l.Panes {
		tab = max(tab, p.Rect.X+p.Rect.Width)
		if p.ID == pane {
			own = p.Rect.Width
		}
	}
	return tab, own, nil
}

// MinNeighbor is the fewest columns the sidebar leaves to the pane it shares its split with.
const MinNeighbor = 20

// step is a split side by side on the way from the top of the tab to the sidebar: its path (true is the second
// child), whether the sidebar is under its first child, and its width and the width of that child, in columns.
type step struct {
	path        []bool
	first       bool
	width, side int
}

// stepsTo are the splits side by side above pane, from the top of the tab down, with their widths from rects.
func stepsTo(n *Node, pane string, rects map[string]Rect, path []bool) ([]step, bool) {
	if n == nil {
		return nil, false
	}
	if n.Type == "pane" {
		return nil, n.Pane == pane
	}
	for i, child := range []*Node{n.First, n.Second} {
		below, ok := stepsTo(child, pane, rects, append(path[:len(path):len(path)], i == 1))
		if !ok {
			continue
		}
		if n.Direction != "right" {
			return below, true
		}
		return append([]step{{path: path, first: i == 0, width: span(n, rects), side: span(child, rects)}}, below...), true
	}
	return nil, false
}

// span is the width of the panes under n.
func span(n *Node, rects map[string]Rect) int {
	lo, hi := -1, 0
	var walk func(*Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		if r, ok := rects[n.Pane]; ok && n.Type == "pane" {
			if lo < 0 || r.X < lo {
				lo = r.X
			}
			hi = max(hi, r.X+r.Width)
		}
		walk(n.First)
		walk(n.Second)
	}
	walk(n)
	return hi - max(lo, 0)
}

// Resize makes pane cols columns wide. The split it shares with its neighbor takes the change; when that split is
// too narrow to leave the neighbor MinNeighbor columns, the splits above it grow too.
func Resize(c Client, pane string, cols int) error {
	tab, err := tabOf(c, pane)
	if err != nil {
		return err
	}
	l, err := c.LayoutOf(pane)
	if err != nil {
		return err
	}
	rects := map[string]Rect{}
	for _, p := range l.Panes {
		rects[p.ID] = p.Rect
	}
	var r struct {
		Layout struct {
			Root Node `json:"root"`
		} `json:"layout"`
	}
	if err := c.Call("layout.export", map[string]string{"tab_id": tab}, &r); err != nil {
		return err
	}
	steps, _ := stepsTo(&r.Layout.Root, pane, rects, []bool{})
	if len(steps) == 0 {
		return errors.New("herdr: the sidebar shares no split side by side")
	}
	// from the sidebar up: the width each split must have, until one already has room
	want := make([]int, len(steps))
	top, need := len(steps)-1, cols
	for ; top >= 0; top-- {
		want[top] = need
		s := steps[top]
		if s.width-need >= MinNeighbor || top == 0 {
			break
		}
		need += MinNeighbor
	}
	top = max(top, 0)
	width := steps[top].width
	for i := top; i < len(steps); i++ {
		s := steps[i]
		// a quarter of a column off the edge gives the width whether herdr rounds the split or cuts it
		ratio := (float64(want[i]) + 0.25) / float64(width)
		if !s.first {
			ratio = 1 - (float64(want[i])-0.25)/float64(width)
		}
		ratio = min(max(ratio, 0.02), 0.98)
		if err := c.Call("layout.set_split_ratio", map[string]any{"tab_id": tab, "path": s.path, "ratio": ratio}, nil); err != nil {
			return err
		}
		width = want[i]
	}
	return nil
}

// Sizer keeps the pane of the sidebar at its width while it runs.
type Sizer struct {
	C      Client
	Pane   string
	S      Settings
	screen int // the width of herdr at the last Fit
}

// Fit puts the sidebar at its width: the one of the normal mode (normal, when fit_normal is on), else the one of the
// screen. force applies it now (a change of mode or of a setting); otherwise it applies when the screen changed
// width and adapt is on, and the sidebar only goes back to its minimum when it is narrower, so a width dragged by
// hand stays.
func (z *Sizer) Fit(normal, force bool) error {
	screen, own, err := widths(z.C, z.Pane)
	if err != nil {
		return err
	}
	changed := z.screen != 0 && screen != z.screen
	z.screen = screen
	want, least := z.S.WidthFor(screen), z.S.MinColumns
	if normal && z.S.FitNormal {
		want, least = NormalColumns, NormalColumns
	}
	switch {
	case force || changed && z.S.Adapt:
	case own < least:
		want = least
	default:
		return nil
	}
	if own == want {
		return nil
	}
	return Resize(z.C, z.Pane, want)
}
