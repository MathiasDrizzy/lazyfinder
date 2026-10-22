package herdr

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

// treeHerdr is a herdr with one tab laid out as a tree of splits, width columns wide: it answers pane.list,
// pane.layout and layout.export from the tree, and layout.set_split_ratio changes it, as herdr does.
type treeHerdr struct {
	mu    sync.Mutex
	root  *Node
	width int
}

func pane(id string) *Node { return &Node{Type: "pane", Pane: id} }

func split(ratio float64, first, second *Node) *Node {
	return &Node{Type: "split", Direction: "right", Ratio: ratio, First: first, Second: second}
}

// widths are the widths of the panes, as herdr rounds them.
func (h *treeHerdr) widths() map[string]int {
	out := map[string]int{}
	var walk func(n *Node, x, w int)
	walk = func(n *Node, x, w int) {
		if n.Type == "pane" {
			out[n.Pane] = w
			return
		}
		a := int(n.Ratio * float64(w))
		walk(n.First, x, a)
		walk(n.Second, x+a, w-a)
	}
	walk(h.root, 0, h.width)
	return out
}

func (h *treeHerdr) serve(t *testing.T) Client {
	return Client{Socket: fakeServer(t, func(method string, params json.RawMessage) string {
		h.mu.Lock()
		defer h.mu.Unlock()
		switch method {
		case "pane.list":
			var panes []Pane
			for id := range h.widths() {
				panes = append(panes, Pane{ID: id, Tab: "w1:t1", Workspace: "w1"})
			}
			b, _ := json.Marshal(map[string]any{"panes": panes})
			return `"result":` + string(b)
		case "pane.layout":
			var l Layout
			x := 0
			var walk func(n *Node, w int)
			walk = func(n *Node, w int) {
				if n.Type == "pane" {
					l.Panes = append(l.Panes, LayoutPane{ID: n.Pane, Rect: Rect{X: x, Width: w, Height: 40}})
					x += w
					return
				}
				a := int(n.Ratio * float64(w))
				walk(n.First, a)
				walk(n.Second, w-a)
			}
			walk(h.root, h.width)
			b, _ := json.Marshal(map[string]any{"layout": l})
			return `"result":` + string(b)
		case "layout.export":
			b, _ := json.Marshal(map[string]any{"layout": map[string]any{"tab_id": "w1:t1", "root": h.root}})
			return `"result":` + string(b)
		case "layout.set_split_ratio":
			var p struct {
				Path  []bool  `json:"path"`
				Ratio float64 `json:"ratio"`
			}
			_ = json.Unmarshal(params, &p)
			n := h.root
			for _, second := range p.Path {
				if second {
					n = n.Second
				} else {
					n = n.First
				}
			}
			n.Ratio = p.Ratio
		}
		return `"result":{"type":"ok"}`
	})}
}

func (h *treeHerdr) widthOf(id string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.widths()[id]
}

// Mathias, 2026-10-22: "hay que crear una configuracion para que se quede de un tamaño para que todas abran del mismo
// tamaño". The sidebar takes its width in columns wherever its split is: at the edge of the tab, inside a split, or on
// the right.
func TestResizeGivesTheSidebarItsColumns(t *testing.T) {
	cases := map[string]*Node{
		"left of the tab":     split(0.5, pane("side"), split(0.5, pane("a"), pane("b"))),
		"inside a split":      split(0.5, split(0.5, pane("side"), pane("a")), pane("b")),
		"right of the tab":    split(0.5, pane("a"), pane("side")),
		"right inside splits": split(0.3, pane("a"), split(0.6, pane("b"), pane("side"))),
	}
	for name, root := range cases {
		t.Run(strings.ReplaceAll(name, " ", "-"), func(t *testing.T) {
			h := &treeHerdr{root: root, width: 175}
			if err := Resize(h.serve(t), "side", 30); err != nil {
				t.Fatal(err)
			}
			if got := h.widthOf("side"); got != 30 {
				t.Errorf("width %d, want 30", got)
			}
		})
	}
}

var sized = Settings{Dock: "left", AutoOpen: true, Columns: 30, ExternalColumns: 40, ExternalFrom: 200, Adapt: true, MinColumns: 22, FitNormal: true}

// The sidebar opens by splitting the pane at the edge, so its split can be too narrow for it: the split around it
// grows and the pane next to it keeps room to work (seen in herdr 0.9.5, a neighbor left 3 columns wide).
func TestResizeWidensASplitTooNarrowForTheSidebar(t *testing.T) {
	h := &treeHerdr{root: split(0.2, split(0.9, pane("side"), pane("a")), pane("b")), width: 175}
	if err := Resize(h.serve(t), "side", 30); err != nil {
		t.Fatal(err)
	}
	if side, a := h.widthOf("side"), h.widthOf("a"); side != 30 || a < MinNeighbor {
		t.Fatalf("side %d, neighbor %d", side, a)
	}
}

// Mathias, 2026-10-22: "otra configuracion que es para monitores, ya que puedes ir con un mac y conectarlo con un
// monitor y se va a desconfigurar ... debe ser automatico". When herdr changes width the sidebar takes the width of
// that screen: herdr at least external_from columns wide is an external monitor.
func TestTheSidebarTakesTheWidthOfEachScreen(t *testing.T) {
	h := &treeHerdr{root: split(30.0/175, pane("side"), pane("a")), width: 175}
	z := &Sizer{C: h.serve(t), Pane: "side", S: sized}
	steps := []struct {
		screen, want int
	}{{175, 30}, {250, 40}, {175, 30}}
	for _, s := range steps {
		h.mu.Lock()
		h.width = s.screen
		h.mu.Unlock()
		if err := z.Fit(false, false); err != nil {
			t.Fatal(err)
		}
		if got := h.widthOf("side"); got != s.want {
			t.Fatalf("on %d columns: width %d, want %d", s.screen, got, s.want)
		}
	}
}

// A width dragged by hand stays until the screen changes; below the minimum the sidebar goes back to it ("debemos
// darle un minimo para que no se pueda encoger mas").
func TestADraggedWidthStaysAboveTheMinimum(t *testing.T) {
	h := &treeHerdr{root: split(30.0/175, pane("side"), pane("a")), width: 175}
	z := &Sizer{C: h.serve(t), Pane: "side", S: sized}
	if err := z.Fit(false, false); err != nil {
		t.Fatal(err)
	}
	for _, drag := range []struct{ to, want int }{{50, 50}, {12, 22}} {
		h.mu.Lock()
		h.root.Ratio = float64(drag.to) / 175
		h.mu.Unlock()
		if err := z.Fit(false, false); err != nil {
			t.Fatal(err)
		}
		if got := h.widthOf("side"); got != drag.want {
			t.Fatalf("dragged to %d: width %d, want %d", drag.to, got, drag.want)
		}
	}
}

// With adapt off a new screen keeps the width herdr gives it, as long as it is not below the minimum.
func TestAdaptOffLeavesTheWidthToHerdr(t *testing.T) {
	h := &treeHerdr{root: split(30.0/175, pane("side"), pane("a")), width: 175}
	s := sized
	s.Adapt = false
	z := &Sizer{C: h.serve(t), Pane: "side", S: s}
	if err := z.Fit(false, false); err != nil {
		t.Fatal(err)
	}
	h.mu.Lock()
	h.width = 250
	h.mu.Unlock()
	if err := z.Fit(false, false); err != nil {
		t.Fatal(err)
	}
	ratio := 30.0 / 175
	if got, want := h.widthOf("side"), int(ratio*250); got != want {
		t.Fatalf("width %d, want %d", got, want)
	}
}

// Mathias, 2026-10-22: "al apretar de salir del modo lateral ... que el recuadro se acomode con el minimo para
// funcionar". The normal mode gets the columns it needs, and the sidebar mode its width back.
func TestTheNormalModeGetsTheWidthItNeeds(t *testing.T) {
	h := &treeHerdr{root: split(30.0/175, pane("side"), pane("a")), width: 175}
	z := &Sizer{C: h.serve(t), Pane: "side", S: sized}
	if err := z.Fit(true, true); err != nil {
		t.Fatal(err)
	}
	if got := h.widthOf("side"); got != NormalColumns {
		t.Fatalf("normal mode: width %d, want %d", got, NormalColumns)
	}
	if err := z.Fit(false, true); err != nil {
		t.Fatal(err)
	}
	if got := h.widthOf("side"); got != 30 {
		t.Fatalf("back in the sidebar mode: width %d", got)
	}
	s := sized
	s.FitNormal = false
	z = &Sizer{C: z.C, Pane: "side", S: s}
	if err := z.Fit(true, true); err != nil {
		t.Fatal(err)
	}
	if got := h.widthOf("side"); got != 30 {
		t.Fatalf("fit_normal off: width %d", got)
	}
}

func TestTheWidthOfAScreen(t *testing.T) {
	for screen, want := range map[int]int{120: 30, 199: 30, 200: 40, 400: 40} {
		if got := sized.WidthFor(screen); got != want {
			t.Errorf("%d columns: %d, want %d", screen, got, want)
		}
	}
	s := sized
	s.Columns, s.MinColumns = 22, 26
	if got := s.WidthFor(100); got != 26 {
		t.Errorf("below the minimum: %d", got)
	}
}
