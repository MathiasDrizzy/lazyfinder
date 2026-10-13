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

// noteVisit moves dir (a folder you went to or a file you opened) to the front of the recents and saves the list. A failure to save is not worth
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
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		path := p
		if st.IsDir() {
			items = append(items, pickItem{
				label: filepath.Base(path), hint: m.homeShort(filepath.Dir(path)),
				run: func() tea.Cmd { return m.enter(path, "") },
			})
			continue
		}
		// an opened file: its folder, with the file selected
		items = append(items, pickItem{
			label: filepath.Base(path), hint: m.homeShort(filepath.Dir(path)),
			run: func() tea.Cmd { return m.enter(filepath.Dir(path), filepath.Base(path)) },
		})
	}
	if len(items) == 0 {
		m.setStatus(i18n.T("Aún no hay carpetas recientes", "No recent folders yet"))
		return nil
	}
	m.openPicker(i18n.T("Archivos y carpetas recientes", "Recent files and folders"), items)
	return nil
}
