package ui

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

const searchesFile = "searches.json"

// savedSearch is a query worth keeping, with the scope it ran in.
type savedSearch struct {
	Query     string `json:"query"`
	Recursive bool   `json:"recursive"`
}

// saveSearch keeps the search being typed. An empty query is not saved and the same query is kept once.
func (m *Model) saveSearch() tea.Cmd {
	q := strings.TrimSpace(m.search.input)
	if q == "" {
		return nil
	}
	s := savedSearch{Query: q, Recursive: m.search.recursive}
	if !slices.Contains(m.saved, s) {
		m.saved = append(m.saved, s)
		saveState(m.stateFile(searchesFile), m.saved)
	}
	m.setStatus(i18n.T("Búsqueda guardada", "Search saved"))
	return nil
}

func (m *Model) savedItems() []pickItem {
	items := make([]pickItem, len(m.saved))
	for i, s := range m.saved {
		s := s
		hint := i18n.T("en esta carpeta", "this folder")
		if s.Recursive {
			hint = i18n.T("con subcarpetas", "with subfolders")
		}
		items[i] = pickItem{label: s.Query, hint: hint, run: func() tea.Cmd { return m.runSaved(s) }, del: func() {
			m.saved = slices.DeleteFunc(m.saved, func(x savedSearch) bool { return x == s })
			saveState(m.stateFile(searchesFile), m.saved)
		}}
	}
	return items
}

func (m *Model) showSaved() tea.Cmd {
	if len(m.saved) == 0 {
		m.setStatus(i18n.T("No hay búsquedas guardadas: pulsa ctrl+s en la barra de búsqueda", "No saved searches: press ctrl+s in the search bar"))
		return nil
	}
	m.openPicker(i18n.T("Búsquedas guardadas", "Saved searches"), m.savedItems())
	m.picker.rebuild = m.savedItems
	return nil
}

func (m *Model) runSaved(s savedSearch) tea.Cmd {
	m.cancelSearch()
	m.openSearch()
	m.search.input, m.search.recursive = s.Query, s.Recursive
	m.search.mode = searchResults
	return m.runSearch()
}
