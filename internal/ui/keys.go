package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
)

var sortCycle = []fsmodel.Column{fsmodel.ColName, fsmodel.ColModified, fsmodel.ColCreated, fsmodel.ColSize, fsmodel.ColKind}

func (m *Model) onKey(k tea.KeyPressMsg) tea.Cmd {
	key := k.String()
	if key == "ctrl+c" {
		return tea.Quit
	}
	m.setStatus("")
	switch {
	case m.menu.open:
		return m.menuKey(key)
	case m.pv.open:
		return m.previewKey(key)
	case m.focus == focusSidebar:
		return m.sidebarKey(key)
	}
	return m.listKey(key)
}

func (m *Model) listKey(key string) tea.Cmd {
	page := m.listRows()
	switch key {
	case "q":
		return tea.Quit
	case "up", "k":
		return m.setSelection(m.sel - 1)
	case "down", "j":
		return m.setSelection(m.sel + 1)
	case "pgup":
		return m.setSelection(m.sel - page)
	case "pgdown":
		return m.setSelection(m.sel + page)
	case "home", "g":
		return m.setSelection(0)
	case "end", "G":
		return m.setSelection(len(m.entries()) - 1)
	case "enter", "right", "l":
		return m.openSelected()
	case "backspace", "left", "h":
		return m.parent()
	case "alt+left", "[":
		return m.step(m.hist.Back())
	case "alt+right", "]":
		return m.step(m.hist.Forward())
	case "s":
		return m.cycleSort(false)
	case "S":
		return m.cycleSort(true)
	case ".":
		m.hidden = !m.hidden
		return m.reload()
	case "r":
		return m.reload()
	case "tab":
		m.focus = focusSidebar
	case "space", " ":
		return m.openPreview()
	case "O":
		return m.openMenu()
	case "Q":
		return m.quickLook()
	}
	return nil
}

func (m *Model) cycleSort(reverse bool) tea.Cmd {
	s := m.sort
	if reverse {
		s.Descending = !s.Descending
		return m.resort(s)
	}
	for i, c := range sortCycle {
		if c == s.By {
			s.By = sortCycle[(i+1)%len(sortCycle)]
			break
		}
	}
	s.Descending = false
	return m.resort(s)
}

func (m *Model) sidebarKey(key string) tea.Cmd {
	dests := destinations(m.places())
	switch key {
	case "q":
		return tea.Quit
	case "up", "k":
		m.sideSel = max(0, m.sideSel-1)
	case "down", "j":
		m.sideSel = min(len(dests)-1, m.sideSel+1)
	case "tab", "esc":
		m.focus = focusList
	case "enter", "right", "l":
		if len(dests) == 0 {
			return nil
		}
		return m.goPlace(m.places()[dests[min(m.sideSel, len(dests)-1)]])
	}
	return nil
}

func (m *Model) goPlace(p place) tea.Cmd {
	if p.missing {
		m.setStatus(p.name + " is not available: " + p.path)
		return nil
	}
	m.focus = focusList
	return m.enter(p.path, "")
}
