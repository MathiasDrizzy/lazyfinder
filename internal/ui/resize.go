package ui

import (
	"maps"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/charmbracelet/x/ansi"
)

const resizeStep = 2 // cells a key moves a border

type dragKind int

const (
	dragNone dragKind = iota
	dragSidebar
	dragColumn
	dragPreview
)

// dragState is a border being dragged with the mouse: the size is relative to where the drag began, so the border
// never jumps under the pointer.
type dragState struct {
	kind   dragKind
	col    fsmodel.Column
	startX int
	invert bool // the border of Name: dragging left widens the column on its right
	start  int  // the width of what is resized when the drag began
}

func isResizeAction(a keymap.Action) bool {
	return a == keymap.SidebarNarrow || a == keymap.SidebarWiden || a == keymap.ColumnNarrow || a == keymap.ColumnWiden
}

// setSidebarWidth keeps the sidebar between its minimum and half the window.
func (m *Model) setSidebarWidth(w int) {
	m.deps.Cfg.SidebarWidth = max(config.MinSidebar, min(w, config.MaxSidebar, m.w/2))
}

// setColumnWidth stores the width of an optional column, never so wide that the name gets less than minNameW.
func (m *Model) setColumnWidth(c fsmodel.Column, w int) {
	if c == fsmodel.ColName {
		return
	}
	cur := m.colWidth(c)
	slack := 0
	if specs := m.columns(); len(specs) > 0 {
		slack = specs[0].w - minNameW
	}
	w = max(config.MinColumn, min(w, config.MaxColumn, cur+slack))
	widths := maps.Clone(m.deps.Cfg.ColumnWidths)
	if widths == nil {
		widths = map[string]int{}
	}
	widths[string(c)] = w
	m.deps.Cfg.ColumnWidths = widths
}

// nudgeColumn makes a column wider (delta > 0) or narrower. The name takes what the others leave, so widening it
// narrows the last optional column and the other way round.
func (m *Model) nudgeColumn(c fsmodel.Column, delta int) {
	specs := m.columns()
	if c != fsmodel.ColName {
		m.setColumnWidth(c, m.colWidth(c)+delta)
		return
	}
	if len(specs) < 2 {
		return
	}
	last := specs[len(specs)-1].col
	m.setColumnWidth(last, m.colWidth(last)-delta)
}

func (m *Model) previewWidth() int {
	g := m.millerGeom()
	return g.rightW
}

func (m *Model) setPreviewWidth(w int) {
	m.deps.Cfg.PreviewWidth = max(config.MinPreview, min(w, config.MaxPreview, m.listW()-m.millerGeom().parentW-12))
}

// resizeKey handles < > (sidebar) and ctrl+left, ctrl+right (the column the list is sorted by).
func (m *Model) resizeKey(act keymap.Action) tea.Cmd {
	switch act {
	case keymap.SidebarNarrow:
		m.setSidebarWidth(m.sidebarW() - resizeStep)
		m.saveSize("sidebar_width", m.deps.Cfg.SidebarWidth)
	case keymap.SidebarWiden:
		m.setSidebarWidth(m.sidebarW() + resizeStep)
		m.saveSize("sidebar_width", m.deps.Cfg.SidebarWidth)
	case keymap.ColumnNarrow:
		m.nudgeColumn(m.sort.By, -resizeStep)
		m.saveColumns()
	case keymap.ColumnWiden:
		m.nudgeColumn(m.sort.By, resizeStep)
		m.saveColumns()
	}
	m.ensureVisible()
	return m.refreshPreview()
}

// saveSize remembers one size in config.json; a failure shows in the status bar and the size still holds.
func (m *Model) saveSize(key string, val any) {
	if m.deps.ConfigPath == "" {
		return
	}
	if err := config.Update(m.deps.ConfigPath, map[string]any{key: val}); err != nil {
		m.setError(err)
	}
}

func (m *Model) saveColumns() {
	for c, w := range m.deps.Cfg.ColumnWidths {
		m.saveSize("column_widths."+c, w)
	}
}

// beginDrag starts dragging the border under the pointer.
func (m *Model) beginDrag(z zone, x int) {
	switch z.kind {
	case zoneSideBorder:
		m.drag = dragState{kind: dragSidebar, startX: x, start: m.sidebarW()}
	case zonePreviewBorder:
		m.drag = dragState{kind: dragPreview, startX: x, start: m.previewWidth()}
	case zoneColBorder:
		c := z.col
		if c == fsmodel.ColName {
			if specs := m.columns(); len(specs) > 1 {
				c = specs[1].col
			}
		}
		m.drag = dragState{kind: dragColumn, col: c, startX: x, start: m.colWidth(c), invert: z.col == fsmodel.ColName}
	}
}

func (m *Model) onMotion(msg tea.MouseMotionMsg) tea.Cmd {
	if !m.mouse || m.drag.kind == dragNone {
		return nil
	}
	x := msg.Mouse().X
	switch m.drag.kind {
	case dragSidebar:
		m.setSidebarWidth(m.drag.start + x - m.drag.startX)
	case dragPreview:
		m.setPreviewWidth(m.drag.start - (x - m.drag.startX))
	case dragColumn:
		delta := x - m.drag.startX
		if m.drag.invert {
			delta = -delta
		}
		m.setColumnWidth(m.drag.col, m.drag.start+delta)
	}
	m.ensureVisible()
	return nil
}

// onRelease ends a drag and saves what it changed.
func (m *Model) onRelease(tea.MouseReleaseMsg) tea.Cmd {
	d := m.drag
	m.drag = dragState{}
	switch d.kind {
	case dragSidebar:
		m.saveSize("sidebar_width", m.deps.Cfg.SidebarWidth)
	case dragPreview:
		m.saveSize("preview_width", m.deps.Cfg.PreviewWidth)
	case dragColumn:
		m.saveSize("column_widths."+string(d.col), m.colWidth(d.col))
	}
	return m.refreshPreview()
}

// fitBorder is the double click on a border: the sidebar fits its longest name, a column fits its content and the
// preview goes back to its automatic width.
func (m *Model) fitBorder(z zone) tea.Cmd {
	switch z.kind {
	case zoneSideBorder:
		longest := 0
		for _, p := range m.places() {
			longest = max(longest, ansi.StringWidth(p.name), ansi.StringWidth(p.title))
		}
		m.setSidebarWidth(longest + 6)
		m.saveSize("sidebar_width", m.deps.Cfg.SidebarWidth)
	case zonePreviewBorder:
		m.deps.Cfg.PreviewWidth = 0
		m.saveSize("preview_width", 0)
	case zoneColBorder:
		c := z.col
		if c == fsmodel.ColName {
			m.fitName()
			break
		}
		w := ansi.StringWidth(columnLabel(c))
		for _, e := range m.entries() {
			w = max(w, ansi.StringWidth(m.cell(e, c)))
		}
		m.setColumnWidth(c, w)
		m.saveSize("column_widths."+string(c), m.colWidth(c))
	}
	return m.refreshPreview()
}

// fitName makes the Name column as wide as its longest name, taking the difference from the last optional column.
func (m *Model) fitName() {
	specs := m.columns()
	if len(specs) < 2 {
		return
	}
	need := ansi.StringWidth(columnLabel(fsmodel.ColName))
	for _, e := range m.entries() {
		need = max(need, ansi.StringWidth(m.cell(e, fsmodel.ColName))+m.iconW())
	}
	last := specs[len(specs)-1].col
	m.nudgeColumn(fsmodel.ColName, need-specs[0].w)
	m.saveSize("column_widths."+string(last), m.colWidth(last))
}
