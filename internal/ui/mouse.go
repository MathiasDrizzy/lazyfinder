package ui

import (
	"errors"
	"fmt"
	"strings"

	sysclip "github.com/MathiasDrizzy/lazyfinder/internal/clipboard"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"

	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/charmbracelet/x/ansi"

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
	zoneBoxBorder     // the border between Favorites and Locations: drag to resize
	zoneProjectTab    // a tab of the project mode
	zoneAction        // an action icon of the tree (new file, new folder, refresh, collapse)
	zoneMode          // the button that switches between the normal window and the sidebar mode
	zoneScmRow        // a row of the Git tab
	zoneScmBtn        // a button on a row of the Git tab
	zoneBranch        // the branch label: opens the branch menu
)

// zone is what lies under a screen cell.
type zone struct {
	kind zoneKind
	idx  int
	col  fsmodel.Column
	path string
	act  scmAct
}

// hit resolves a screen position on the base layer (popups are handled before). Panel borders are not clickable.
func (m *Model) hit(x, y int) zone {
	lastRow := m.bodyEnd()
	for _, a := range m.actionSpans() {
		if y == a.y && x >= a.x && x < a.x+a.w {
			return zone{kind: zoneAction, idx: a.idx}
		}
	}
	if m.scmOn() {
		if y >= m.firstRow() && y < lastRow && x >= m.listX() && x < m.listX()+m.listW() {
			i := m.scm.top + y - m.firstRow()
			if i < len(m.scmRows()) {
				for _, b := range m.scmButtons(i) {
					if x >= b.x && x < b.x+b.w {
						return zone{kind: zoneScmBtn, idx: i, act: b.act}
					}
				}
				return zone{kind: zoneScmRow, idx: i}
			}
		}
	}
	if m.columnsOn() && y >= m.headerRow() && y < lastRow && x == m.listX()+m.millerGeom().parentW+1+m.millerGeom().centerW {
		return zone{kind: zonePreviewBorder}
	}
	if m.columnsOn() && y >= m.headerRow() && y < lastRow && x >= m.listX() && x < m.listX()+m.listW() {
		return m.millerHit(x, y)
	}
	if l := m.sidebarBoxLayout(); len(l) == 2 && x >= 1 && x < m.sidebarW()-1 && (y == l[0].h || y == l[0].h+1) {
		return zone{kind: zoneBoxBorder}
	}
	if y >= 1 && y < m.panelH() && m.sidebarW() > 0 && (x == m.sidebarW()-1 || x == m.sidebarW()) {
		return zone{kind: zoneSideBorder}
	}
	if !m.columnsOn() && !m.projectMode() && y == m.headerRow() {
		for _, s := range m.columns() {
			if rel := x - m.listX(); rel >= s.x+s.w && rel < s.x+s.w+m.gap() {
				return zone{kind: zoneColBorder, col: s.col}
			}
		}
	}
	switch {
	case y < activityH && m.projectMode(): // every row of a tab bar button takes the click, as in herdr
		if t, ok := m.projectTabAt(x); ok {
			return zone{kind: zoneProjectTab, idx: int(t)}
		}
		if x >= m.gearX() {
			return zone{kind: zoneSettings}
		}
		if x >= m.modeX() && x < m.modeX()+m.chipW() {
			return zone{kind: zoneMode}
		}
	case y == 0:
		bx, bw, hasBranch := m.branchSpan()
		switch {
		case hasBranch && x >= bx && x < bx+bw:
			return zone{kind: zoneBranch}
		case x < 3:
			return zone{kind: zoneBack}
		case x < 6:
			return zone{kind: zoneForward}
		case x >= m.gearX():
			return zone{kind: zoneSettings}
		case x >= m.modeX() && x < m.modeX()+m.chipW():
			return zone{kind: zoneMode}
		case x >= m.fieldX() && x < m.fieldX()+m.fieldW():
			return zone{kind: zoneSearch}
		}
	case y == m.h-2 && m.projectMode() && m.git.ok && x < ansi.StringWidth(strings.TrimRight(ansi.Strip(m.projectFooter()), " ")):
		return zone{kind: zoneBranch}
	case y == m.h-1 && m.footerShowsHints():
		_, spans := m.footerHints()
		for _, s := range spans {
			if x >= s.x && x < s.x+s.w {
				return zone{kind: zoneHint, path: string(s.action)}
			}
		}
	case y == m.h-1 && m.searchOn():
		return zone{kind: zoneSearch}
	case y == m.h-2 && m.searchOn():
		_, spans := m.filterBar()
		for i, s := range spans {
			if x >= s.x && x < s.x+s.w {
				return zone{kind: zoneChip, idx: i}
			}
		}
	case y == m.h-2 && !m.projectMode():
		for _, c := range m.crumbs() {
			if x >= c.x && x < c.x+c.w {
				return zone{kind: zoneCrumb, path: c.path}
			}
		}
	case y >= m.headerRow() && y < lastRow && x >= 1 && x < m.sidebarW()-1:
		return m.sidebarHit(y - 1)
	case y == m.headerRow() && !m.projectMode() && x >= m.listX() && x < m.listX()+m.listW():
		for _, s := range m.columns() {
			if x-m.listX() >= s.x && x-m.listX() < s.x+s.w {
				return zone{kind: zoneHeader, col: s.col}
			}
		}
	case y >= m.firstRow() && y < lastRow && x >= m.listX() && x < m.listX()+m.listW():
		if idx := m.top + (y-m.firstRow())/m.rowH(); idx < len(m.entries()) {
			if x == m.listX()+2 && m.entries()[idx].IsDir && !m.searchOn() && !m.projectMode() { // the > marker
				return zone{kind: zoneToggle, idx: idx}
			}
			return zone{kind: zoneRow, idx: idx}
		}
	}
	return zone{}
}

func (m *Model) sidebarHit(row int) zone {
	ps := m.places()
	pi, ok := m.sidebarItemAt(row)
	if !ok {
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
	if m.diff.open {
		return nil // like every popup, it closes only with Esc
	}
	if m.gm.open {
		return m.gitMenuClick(mouse.X, mouse.Y)
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
		m.closeInfo()
		return nil
	}
	if m.pv.open {
		if x, y, bx := m.previewBox(); !inBox(mouse.X, mouse.Y, x, y, bx) {
			return m.closePreview()
		}
		return nil
	}
	m.pointerX, m.pointerY = mouse.X, mouse.Y
	z := m.hit(mouse.X, mouse.Y)
	if mouse.Button == tea.MouseRight && m.openContext(z, mouse.X, mouse.Y) {
		return nil
	}
	if mouse.Button == tea.MouseRight && z.kind == zoneCrumb {
		return m.copyPath(z.path)
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
	case zoneSideBorder, zoneColBorder, zonePreviewBorder, zoneBoxBorder:
		if prev.zone == z && m.deps.Now().Sub(prev.at) <= doubleClickWindow {
			m.lastClick = click{}
			m.drag = dragState{}
			return m.fitBorder(z)
		}
		m.beginDrag(z, m.pointerX)
		return nil
	case zoneHint:
		return m.listKey(keymap.Action(z.path))
	case zoneProjectTab:
		return m.clickProjectTab(projectTab(z.idx))
	case zoneAction:
		return m.runAction(z.idx)
	case zoneMode:
		m.toggleProject()
		return nil
	case zoneBranch:
		return m.openGitMenu()
	case zoneScmBtn:
		return m.scmClickBtn(z.idx, z.act)
	case zoneScmRow:
		double := prev.zone.kind == zoneScmRow && prev.zone.idx == z.idx && m.deps.Now().Sub(prev.at) <= doubleClickWindow
		if double {
			m.lastClick = click{}
		}
		return m.scmClickRow(z.idx, double)
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
		if m.projectMode() && !m.searchOn() && z.idx < len(m.entries()) && m.entries()[z.idx].IsDir {
			m.sel = z.idx // in the tree a click anywhere on a folder opens or closes it, like the herdr sidebar
			m.lastClick = click{}
			return m.toggle(m.entries()[z.idx].Path)
		}
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
	if m.diff.open {
		m.diff.scroll = max(0, min(m.diffMax(), m.diff.scroll+dir*wheelStep))
		return nil
	}
	if m.gm.open {
		return nil
	}
	if m.pv.open {
		m.scrollPreview(dir * wheelStep)
		return nil
	}
	if m.scmOn() {
		m.scm.top = max(0, min(max(0, len(m.scmRows())-m.scmVisible()), m.scm.top+dir*wheelStep))
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

// copyPath puts a folder of the breadcrumb on the clipboard and says so.
func (m *Model) copyPath(path string) tea.Cmd { return m.copyText(path, path) }

// copyText puts text on the clipboard and says it copied label. Without a clipboard tool, the terminal sets it
// (OSC 52).
func (m *Model) copyText(text, label string) tea.Cmd {
	if m.deps.Copy == nil {
		return nil
	}
	var cmd tea.Cmd
	switch err := m.deps.Copy(text); {
	case errors.Is(err, sysclip.ErrNoTool):
		cmd = tea.SetClipboard(text)
	case err != nil:
		m.setError(fmt.Errorf(i18n.T("No se pudo copiar la ruta: %w", "Cannot copy the path: %w"), err))
		return nil
	}
	m.setStatus(fmt.Sprintf(i18n.T("Copiado: %s", "Copied %s"), label))
	return cmd
}
