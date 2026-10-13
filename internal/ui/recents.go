package ui

import (
	"os"
	"path/filepath"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

const (
	maxRecents  = 20
	recentsFile = "recents.json"
)

// noteVisit moves dir to the front of the recent folders and saves the list. A failure to save is not worth
// interrupting navigation for: the list still works for this session.
func (m *Model) noteVisit(dir string) {
	m.recents = slices.DeleteFunc(m.recents, func(p string) bool { return p == dir })
	m.recents = append([]string{dir}, m.recents...)
	m.recents = m.recents[:min(len(m.recents), maxRecents)]
	saveState(m.stateFile(recentsFile), m.recents)
}

func (m *Model) showRecents() tea.Cmd {
	var items []pickItem
	for _, p := range m.recents {
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			continue
		}
		dir := p
		items = append(items, pickItem{
			label: filepath.Base(dir), hint: m.homeShort(filepath.Dir(dir)),
			run: func() tea.Cmd { return m.enter(dir, "") },
		})
	}
	if len(items) == 0 {
		m.setStatus(i18n.T("Aún no hay carpetas recientes", "No recent folders yet"))
		return nil
	}
	m.openPicker(i18n.T("Carpetas recientes", "Recent folders"), items)
	return nil
}
