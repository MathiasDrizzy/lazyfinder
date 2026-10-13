package ui

import (
	"fmt"
	"maps"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/history"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// tabState is what one tab remembers while another is shown: where it is and how it looks. Everything else
// (clipboard, undo, marks, the open popups) belongs to the window.
type tabState struct {
	hist     history.History
	sort     fsmodel.Sort
	hidden   bool
	expanded map[string]bool
	sel      string // name of the selected top-level row
}

func (m *Model) snapshot() tabState {
	t := tabState{hist: m.hist, sort: m.sort, hidden: m.hidden, expanded: maps.Clone(m.expanded)}
	if e, ok := m.selected(); ok && e.Depth == 0 {
		t.sel = e.Name
	}
	return t
}

func (m *Model) tabCount() int { return max(1, len(m.tabs)) }

// tabLabel is shown in the toolbar when there is more than one tab.
func (m *Model) tabLabel() string {
	if m.tabCount() < 2 {
		return ""
	}
	return fmt.Sprintf(i18n.T("Pestaña %d/%d", "Tab %d/%d"), m.cur+1, m.tabCount())
}

func (m *Model) ensureTabs() {
	if len(m.tabs) == 0 {
		m.tabs, m.cur = []tabState{m.snapshot()}, 0
	}
}

// newTab opens a copy of the current tab right after it and shows it.
func (m *Model) newTab() tea.Cmd {
	m.ensureTabs()
	m.tabs[m.cur] = m.snapshot()
	at := m.cur + 1
	m.tabs = append(m.tabs[:at], append([]tabState{m.snapshot()}, m.tabs[at:]...)...)
	m.cur = at
	m.setStatus("")
	return nil
}

// switchTab shows tab i (wrapping around).
func (m *Model) switchTab(i int) tea.Cmd {
	if m.tabCount() < 2 {
		return nil
	}
	m.tabs[m.cur] = m.snapshot()
	return m.showTab((i%len(m.tabs) + len(m.tabs)) % len(m.tabs))
}

func (m *Model) showTab(i int) tea.Cmd {
	m.cancelSearch()
	t := m.tabs[i]
	prev := m.listing
	m.hist, m.sort, m.hidden, m.expanded = t.hist, t.sort, t.hidden, maps.Clone(t.expanded)
	if m.expanded == nil {
		m.expanded = map[string]bool{}
	}
	if err := m.load(m.dir(), t.sel); err != nil {
		m.listing = prev
		m.setError(err)
	}
	m.cur = i
	m.fresh = map[string]bool{}
	m.follow()
	return m.refreshPreview()
}

func (m *Model) closeTab() tea.Cmd {
	if m.tabCount() < 2 {
		m.setStatus(i18n.T("Es la última pestaña", "This is the last tab"))
		return nil
	}
	m.tabs = append(m.tabs[:m.cur], m.tabs[m.cur+1:]...)
	return m.showTab(min(m.cur, len(m.tabs)-1))
}

func (m *Model) cancelSearch() {
	if m.search.mode == searchOff {
		return
	}
	if m.search.cancel != nil {
		m.search.cancel()
	}
	m.search = searchState{}
}
