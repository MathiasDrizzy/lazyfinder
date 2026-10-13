package ui

import (
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"

	tea "charm.land/bubbletea/v2"
)

type zoneKind int

const (
	zoneNone zoneKind = iota
	zoneBack
	zoneForward
	zonePlace
	zoneRow
	zoneHeader
	zoneCrumb
	zoneSettings
	zoneSearch
	zoneChip
	zoneToggle
	zoneHint          // a key hint of the bottom row
	zoneSideBorder    // the border between the sidebar and the list: drag to resize
	zoneColBorder     // the gap right of a column header
	zonePreviewBorder // the border in front of the preview in the column view
)

// zone is what lies under a screen cell.
type zone struct {
	kind zoneKind
	idx  int
	col  fsmodel.Column
	path string
}

// hit resolves a screen position on the base layer (popups are handled before). Panel borders are not clickable.
func (m *Model) hit(x, y int) zone {
	lastRow := m.panelH() - 1 // last panel row (its bottom border)
	if m.columnsOn() && y >= headerRow && y < lastRow && x == m.listX()+m.millerGeom().parentW+1+m.millerGeom().centerW {
		return zone{kind: zonePreviewBorder}
	}
	if m.columnsOn() && y >= headerRow && y < lastRow && x >= m.listX() && x < m.listX()+m.listW() {
		return m.millerHit(x, y)
	}
	if y >= 1 && y <= lastRow && m.sidebarW() > 0 && (x == m.sidebarW()-1 || x == m.sidebarW()) {
		return zone{kind: zoneSideBorder}
	}
	if !m.columnsOn() && y == headerRow {
		for _, s := range m.columns() {
			if rel := x - m.listX(); rel >= s.x+s.w && rel < s.x+s.w+colGap {
				return zone{kind: zoneColBorder, col: s.col}
			}
		}
	}
	switch {
	case y == 0:
		switch {
		case x < 3:
			return zone{kind: zoneBack}
		case x < 6:
			return zone{kind: zoneForward}
		case x >= m.w-4:
			return zone{kind: zoneSettings}
		case x >= m.w-4-m.fieldW() && x < m.w-4:
			return zone{kind: zoneSearch}
		}
	case y == m.h-1 && m.footerShowsHints():
		_, spans := m.footerHints()
		for _, s := range spans {
			if x >= s.x && x < s.x+s.w {
				return zone{kind: zoneHint, path: string(s.action)}
			}
		}
	case y == m.h-1 && m.searchOn():
		_, spans := m.searchBar()
		for i, s := range spans {
			if x >= s.x && x < s.x+s.w {
				return zone{kind: zoneChip, idx: i}
			}
		}
	case y == m.h-2:
		for _, c := range m.crumbs() {
			if x >= c.x && x < c.x+c.w {
				return zone{kind: zoneCrumb, path: c.path}
			}
		}
	case y >= headerRow && y < lastRow && x >= 1 && x < m.sidebarW()-1:
		return m.sidebarHit(y - headerRow)
	case y == headerRow && x >= m.listX() && x < m.listX()+m.listW():
		for _, s := range m.columns() {
			if x-m.listX() >= s.x && x-m.listX() < s.x+s.w {
				return zone{kind: zoneHeader, col: s.col}
			}
		}
	case y >= firstRow && y < lastRow && x >= m.listX() && x < m.listX()+m.listW():
		if idx := m.top + (y-firstRow)/m.rowH(); idx < len(m.entries()) {
			if x == m.listX()+2 && m.entries()[idx].IsDir && !m.searchOn() { // the > marker
				return zone{kind: zoneToggle, idx: idx}
			}
			return zone{kind: zoneRow, idx: idx}
		}
	}
	return zone{}
}

func (m *Model) sidebarHit(pi int) zone {
	ps := m.places()
	if pi >= len(ps) || ps[pi].title != "" || ps[pi].name == "" {
		return zone{}
	}
	dests := destinations(ps)
	for di, d := range dests {
		if d == pi {
			return zone{kind: zonePlace, idx: di, path: ps[pi].path}
		}
	}
	return zone{}
}

func (m *Model) onClick(c tea.MouseClickMsg) tea.Cmd {
	mouse := c.Mouse()
	if m.tooSmallNow() || !m.mouse {
		return nil
	}
	m.setStatus("")
	if m.ctx.open {
		return m.ctxClick(mouse.X, mouse.Y)
	}
	if m.help.open {
		return nil // like lazymark: a popup closes only with Esc
	}
	if m.set.open {
		return m.settingsClick(mouse.X, mouse.Y)
	}
	if m.picker.open {
		return m.pickerClick(mouse.X, mouse.Y)
	}
	if m.menu.open {
		return m.menuClick(mouse.X, mouse.Y)
	}
	if m.info.open {
		m.info = infoState{}
		return nil
	}
	if m.pv.open {
		if x, y, bx := m.previewBox(); !inBox(mouse.X, mouse.Y, x, y, bx) {
			return m.closePreview()
		}
		return nil
	}
	m.pointerX = mouse.X
	z := m.hit(mouse.X, mouse.Y)
	if mouse.Button == tea.MouseRight && m.openContext(z, mouse.X, mouse.Y) {
		return nil
	}
	if mouse.Button == tea.MouseRight && z.kind == zoneRow {
		m.focus = focusList
		m.sel = z.idx
		return m.openMenu()
	}
	if mouse.Button != tea.MouseLeft {
		return nil
	}
	if z.kind == zoneRow && mouse.Mod&tea.ModCtrl != 0 { // ctrl-click marks the row without opening it
		m.focus = focusList
		m.toggleMark(m.entries()[z.idx].Path)
		return m.setSelection(z.idx)
	}
	return m.clickZone(z)
}

func (m *Model) clickZone(z zone) tea.Cmd {
	prev := m.lastClick
	m.lastClick = click{zone: z, at: m.deps.Now()}
	switch z.kind {
	case zoneSideBorder, zoneColBorder, zonePreviewBorder:
		if prev.zone == z && m.deps.Now().Sub(prev.at) <= doubleClickWindow {
			m.lastClick = click{}
			m.drag = dragState{}
			return m.fitBorder(z)
		}
		m.beginDrag(z, m.pointerX)
		return nil
	case zoneHint:
		return m.listKey(keymap.Action(z.path))
	case zoneBack:
		return m.step(m.hist.Back())
	case zoneForward:
		return m.step(m.hist.Forward())
	case zoneSettings:
		m.set = settingsState{open: true}
		return nil
	case zoneSearch:
		m.openSearch()
		return nil
	case zoneToggle:
		m.focus = focusList
		m.sel = z.idx
		return m.toggle(m.entries()[z.idx].Path)
	case zoneChip:
		return m.clickChip(z.idx)
	case zoneCrumb:
		if z.path == m.dir() {
			return nil
		}
		return m.enter(z.path, "")
	case zonePlace:
		m.sideSel = z.idx
		return m.goPlace(m.places()[destinations(m.places())[z.idx]])
	case zoneHeader:
		s := m.sort
		if s.By == z.col {
			s.Descending = !s.Descending
		} else {
			s.By, s.Descending = z.col, false
		}
		return m.resort(s)
	case zoneRow:
		m.focus = focusList
		double := prev.zone.kind == zoneRow && prev.zone.idx == z.idx && m.deps.Now().Sub(prev.at) <= doubleClickWindow
		if double {
			m.lastClick = click{}
			m.sel = z.idx
			return m.openSelected()
		}
		return m.setSelection(z.idx)
	}
	return nil
}

func (m *Model) onWheel(w tea.MouseWheelMsg) tea.Cmd {
	mouse := w.Mouse()
	if !m.mouse || m.set.open {
		return nil
	}
	dir := 0
	switch mouse.Button {
	case tea.MouseWheelUp:
		dir = -1
	case tea.MouseWheelDown:
		dir = 1
	default:
		return nil
	}
	if m.pv.open {
		m.scrollPreview(dir * wheelStep)
		return nil
	}
	if m.menu.open {
		return nil
	}
	maxTop := max(0, len(m.entries())-m.listRows())
	m.top = max(0, min(maxTop, m.top+dir*wheelStep))
	return nil
}

func inBox(x, y, bx, by int, box []string) bool {
	if len(box) == 0 {
		return false
	}
	return y >= by && y < by+len(box) && x >= bx && x < bx+boxWidth(box)
}
