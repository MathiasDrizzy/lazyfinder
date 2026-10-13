package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
)

var sortCycle = []fsmodel.Column{fsmodel.ColName, fsmodel.ColModified, fsmodel.ColCreated, fsmodel.ColSize, fsmodel.ColKind}

func (m *Model) onKey(k tea.KeyPressMsg) tea.Cmd {
	key := k.String()
	if key == "ctrl+c" {
		return tea.Quit
	}
	m.setStatus("")
	act, _ := m.keys.Lookup(key)
	switch {
	case m.set.open:
		return m.settingsKey(key, act)
	case m.picker.open:
		return m.pickerKey(key, act)
	case m.menu.open:
		return m.menuKey(key, act)
	case m.pv.open:
		return m.previewKey(key, act)
	case m.info.open:
		m.info = infoState{}
		return nil
	case m.prompt.kind != promptNone:
		return m.promptKey(k)
	case m.goto_.open:
		return m.gotoKey(k)
	case m.search.mode == searchTyping:
		return m.searchKey(k)
	case m.search.mode == searchResults:
		if cmd, ok := m.resultsKey(key); ok {
			return cmd
		}
	case m.focus == focusSidebar:
		return m.sidebarKey(key, act)
	}
	if key == "esc" && m.markCount() > 0 {
		clear(m.marked)
		return nil
	}
	if key == "esc" && m.filter != "" {
		return m.setFilter("")
	}
	return m.listKey(act)
}

func (m *Model) listKey(act keymap.Action) tea.Cmd {
	page := m.listRows()
	if m.searchOn() && (act == keymap.Hidden || act == keymap.Ignored || act == keymap.Reload) {
		return nil // the list holds search results, not the folder
	}
	if cmd, ok := m.opsKey(act); ok {
		return cmd
	}
	switch act {
	case keymap.Quit:
		return tea.Quit
	case keymap.Up:
		return m.setSelection(m.sel - 1)
	case keymap.Down:
		return m.setSelection(m.sel + 1)
	case keymap.PageUp:
		return m.setSelection(m.sel - page)
	case keymap.PageDown:
		return m.setSelection(m.sel + page)
	case keymap.First:
		return m.setSelection(0)
	case keymap.Last:
		return m.setSelection(len(m.entries()) - 1)
	case keymap.Open:
		return m.openSelected()
	case keymap.Parent:
		if m.deps.Follow {
			return m.collapseSelected()
		}
		return m.parent()
	case keymap.Back:
		return m.step(m.hist.Back())
	case keymap.Forward:
		return m.step(m.hist.Forward())
	case keymap.Sort:
		return m.cycleSort(false)
	case keymap.SortReverse:
		return m.cycleSort(true)
	case keymap.Hidden:
		m.hidden = !m.hidden
		return m.reload()
	case keymap.Ignored:
		m.hideIgnored = !m.hideIgnored
		return m.reload()
	case keymap.Expand:
		return m.expandSelected()
	case keymap.Collapse:
		return m.collapseSelected()
	case keymap.Layout:
		return m.toggleLayout()
	case keymap.TagKey:
		return m.startTag()
	case keymap.Tags:
		return m.showTags()
	case keymap.SavedSearches:
		return m.showSaved()
	case keymap.Filter:
		return m.startFilter()
	case keymap.Commands:
		return m.showCommands()
	case keymap.Recents:
		return m.showRecents()
	case keymap.Bookmark:
		return m.toggleBookmark()
	case keymap.NewTab:
		return m.newTab()
	case keymap.CloseTab:
		return m.closeTab()
	case keymap.NextTab:
		return m.switchTab(m.cur + 1)
	case keymap.PrevTab:
		return m.switchTab(m.cur - 1)
	case keymap.Goto:
		m.goto_ = gotoState{open: true}
	case keymap.Search:
		m.openSearch()
	case keymap.Reload:
		return m.reload()
	case keymap.Focus:
		m.focus = focusSidebar
	case keymap.Preview:
		return m.openPreview()
	case keymap.OpenWith:
		return m.openMenu()
	case keymap.QuickLook:
		return m.quickLook()
	case keymap.Settings:
		m.set = settingsState{open: true}
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

func (m *Model) sidebarKey(key string, act keymap.Action) tea.Cmd {
	dests := destinations(m.places())
	switch {
	case act == keymap.Quit:
		return tea.Quit
	case act == keymap.Up:
		m.sideSel = max(0, m.sideSel-1)
	case act == keymap.Down:
		m.sideSel = min(len(dests)-1, m.sideSel+1)
	case act == keymap.Focus || key == "esc":
		m.focus = focusList
	case act == keymap.Settings:
		m.set = settingsState{open: true}
	case act == keymap.Open:
		if len(dests) == 0 {
			return nil
		}
		return m.goPlace(m.places()[dests[min(m.sideSel, len(dests)-1)]])
	}
	return nil
}

func (m *Model) goPlace(p place) tea.Cmd {
	if p.missing {
		m.setStatus(fmt.Sprintf(i18n.T("%s no está disponible: %s", "%s is not available: %s"), p.name, p.path))
		return nil
	}
	m.focus = focusList
	return m.enter(p.path, "")
}
